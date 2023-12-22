package grid_tree

import (
	"errors"
	"log"
	"runtime"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/point_loader"
)

// Coordinates are stored in EPSG 3395, which is a cartesian 2D metric reference system
const internalCoordinateEpsgCode = 3395

// Represents an GridTree of points and contains all information needed
// to propagate points in the tree
type GridTree struct {
	rootNode            octree.INode
	built               bool
	maxCellSize         float64
	minCellSize         float64
	coordinateConverter converters.CoordinateConverter
	elevationCorrector  converters.ElevationCorrector
	point_loader.Loader
	sync.RWMutex
}

// Builds an empty GridTree initializing its properties to the correct defaults
func NewGridTree(coordinateConverter converters.CoordinateConverter, elevationCorrector converters.ElevationCorrector, maxCellSize float64, minCellSize float64) octree.ITree {
	return &GridTree{
		built:               false,
		maxCellSize:         maxCellSize,
		minCellSize:         minCellSize,
		Loader:              point_loader.NewSequentialLoader(),
		coordinateConverter: coordinateConverter,
		elevationCorrector:  elevationCorrector,
	}
}

// Builds the hierarchical tree structure
func (tree *GridTree) Build() error {
	if tree.built {
		return errors.New("octree already built")
	}

	tree.init()

	var wg sync.WaitGroup
	tree.launchParallelPointLoaders(&wg)
	wg.Wait()

	tree.rootNode.(*GridNode).BuildPoints()
	tree.built = true

	return nil
}

func (tree *GridTree) GetRootNode() octree.INode {
	return tree.rootNode
}

func (tree *GridTree) IsBuilt() bool {
	return tree.built
}

func (tree *GridTree) AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) {
	tree.Loader.AddPoint(tree.getPointFromRawData(coordinate, r, g, b, intensity, classification, srid))
}

func (tree *GridTree) getPointFromRawData(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) *data.Point {
	wgs84coords, err := tree.coordinateConverter.ConvertCoordinateSrid(srid, 4326, *coordinate)
	z := tree.elevationCorrector.CorrectElevation(wgs84coords.X, wgs84coords.Y, wgs84coords.Z)

	worldMercatorCoords, err := tree.coordinateConverter.ConvertCoordinateSrid(
		srid,
		internalCoordinateEpsgCode,
		geometry.Coordinate{
			X: coordinate.X,
			Y: coordinate.Y,
			Z: z,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	return data.NewPoint(worldMercatorCoords.X, worldMercatorCoords.Y, worldMercatorCoords.Z, r, g, b, intensity, classification)
}

func (tree *GridTree) init() {
	box := tree.GetBounds()
	node := NewGridNode(nil, geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), tree.maxCellSize, tree.minCellSize, true, "[root]")
	tree.rootNode = node
	tree.InitializeLoader()
}

func (tree *GridTree) launchParallelPointLoaders(waitGroup *sync.WaitGroup) {
	N := runtime.NumCPU()

	for i := 0; i < N; i++ {
		waitGroup.Add(1)
		go tree.launchPointLoader(waitGroup)
	}
}

func (tree *GridTree) launchPointLoader(waitGroup *sync.WaitGroup) {
	for {
		val, shouldContinue := tree.Loader.GetNext()
		if val != nil {
			tree.rootNode.AddDataPoint(val)
		}
		if !shouldContinue {
			break
		}
	}
	waitGroup.Done()
}
