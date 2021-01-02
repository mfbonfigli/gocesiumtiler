package random_trees

import (
	"errors"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/point_loader"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"log"
	"math"
	"runtime"
	"sync"
)

// Represents an octTree of points and contains all information needed
// to propagate points in the tree
type octTree struct {
	itemsToAdd                         []data.Point
	rootNode                           octree.INode
	built                              bool
	minX, maxX, minY, maxY, minZ, maxZ float64
	opts                               *tiler.TilerOptions
	coordinateConverter                converters.CoordinateConverter
	elevationCorrector                 converters.ElevationCorrector
	point_loader.Loader
}

// Builds an empty octTree initializing its properties to the correct defaults
func NewRandomTree(opts *tiler.TilerOptions, coordinateConverter converters.CoordinateConverter, elevationCorrector converters.ElevationCorrector) octree.ITree {
	return &octTree{
		itemsToAdd:          make([]data.Point, 0),
		built:               false,
		minX:                math.MaxFloat64,
		minY:                math.MaxFloat64,
		minZ:                math.MaxFloat64,
		maxX:                -1 * math.MaxFloat64,
		maxY:                -1 * math.MaxFloat64,
		maxZ:                -1 * math.MaxFloat64,
		opts:                opts,
		Loader:              point_loader.NewRandomLoader(),
		coordinateConverter: coordinateConverter,
		elevationCorrector:  elevationCorrector,
	}
}

func NewBoxedRandomTree(opts *tiler.TilerOptions, coordinateConverter converters.CoordinateConverter, elevationCorrector converters.ElevationCorrector) octree.ITree {
	return &octTree{
		itemsToAdd:          make([]data.Point, 0),
		built:               false,
		minX:                math.MaxFloat64,
		minY:                math.MaxFloat64,
		minZ:                math.MaxFloat64,
		maxX:                -1 * math.MaxFloat64,
		maxY:                -1 * math.MaxFloat64,
		maxZ:                -1 * math.MaxFloat64,
		opts:                opts,
		Loader:              point_loader.NewRandomBoxLoader(),
		coordinateConverter: coordinateConverter,
		elevationCorrector:  elevationCorrector,
	}
}

// Internally update the bounds of the tree.
// TODO: These could be read directly from the LAS file
func (tree *octTree) recomputeBoundsFromElement(element *data.Point) {
	tree.minX = math.Min(float64(element.X), tree.minX)
	tree.minY = math.Min(float64(element.Y), tree.minY)
	tree.minZ = math.Min(float64(element.Z), tree.minZ)
	tree.maxX = math.Max(float64(element.X), tree.maxX)
	tree.maxY = math.Max(float64(element.Y), tree.maxY)
	tree.maxZ = math.Max(float64(element.Z), tree.maxZ)
}

// Builds the hierarchical tree structure propagating the added items according to the TilerOptions provided
// during initialization
func (tree *octTree) Build() error {
	if tree.built {
		return errors.New("octree already built")
	}

	tree.init()

	var wg sync.WaitGroup
	tree.launchParallelPointLoaders(&wg)
	wg.Wait()

	tree.itemsToAdd = nil
	tree.built = true

	return nil
}

func (tree *octTree) init() {
	box := tree.GetBounds()
	octNode := NewOctNode(geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), tree.opts, 1, nil)
	tree.rootNode = octNode
	tree.InitializeLoader()
}

func (tree *octTree) launchParallelPointLoaders(waitGroup *sync.WaitGroup) {
	N := runtime.NumCPU()

	for i := 0; i < N; i++ {
		waitGroup.Add(1)
		go tree.launchPointLoader(waitGroup)
	}
}

func (tree *octTree) launchPointLoader(waitGroup *sync.WaitGroup) {
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

// Prints the tree structure
func (tree *octTree) PrintStructure() {
	if tree.built {
		tree.rootNode.PrintStructure()
	}
}

func (tree *octTree) GetRootNode() octree.INode {
	return tree.rootNode
}

func (tree *octTree) IsBuilt() bool {
	return tree.built
}

func (tree *octTree) AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) {
	tree.Loader.AddPoint(tree.getPointFromRawData(coordinate, r, g, b, intensity, classification, srid))
}

func (tree *octTree) getPointFromRawData(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) *data.Point {
	tr, err := tree.coordinateConverter.ConvertCoordinateSrid(srid, 4326, *coordinate)
	if err != nil {
		log.Fatal(err)
	}

	return data.NewPoint(*tr.X, *tr.Y, tree.elevationCorrector.CorrectElevation(*tr.X, *tr.Y, *tr.Z), r, g, b, intensity, classification)
}
