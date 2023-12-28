package grid_tree

import (
	"errors"
	"log"
	"math"
	"runtime"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/point_loader"
)

// Coordinates are stored in EPSG 3395, which is a cartesian 2D metric reference system
const internalCoordinateEpsgCode = 3395

// Represents an GridTree of points and contains all information needed
// to propagate points in the tree
type GridTree struct {
	offX, offY, offZ    float64
	offsetInit          bool
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
func (t *GridTree) Build(l las.LasReader) error {
	if t.built {
		return errors.New("octree already built")
	}

	for i := 0; i < int(math.Min(float64(l.NumberOfPoints()), 400000)); i++ {
		x, y, z, r, g, b, in, cls := l.GetPointAt(i)
		t.AddPoint(&geometry.Coordinate{X: x, Y: y, Z: z}, r, g, b, in, cls, l.GetSrid())
	}

	t.init()

	var wg sync.WaitGroup
	t.launchParallelPointLoaders(&wg)
	wg.Wait()

	t.rootNode.(*GridNode).BuildPoints()
	t.built = true

	return nil
}

func (t *GridTree) GetRootNode() octree.INode {
	return t.rootNode
}

func (t *GridTree) IsBuilt() bool {
	return t.built
}

func (t *GridTree) AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) {
	t.Loader.AddPoint(t.getPointFromRawData(coordinate, r, g, b, intensity, classification, srid))
}

func (t *GridTree) getPointFromRawData(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) *data.Point {
	wgs84coords, err := t.coordinateConverter.ConvertCoordinateSrid(srid, 4326, *coordinate)
	if err != nil {
		log.Fatalf("unable to convert coordinate: %v", err)
	}
	z := t.elevationCorrector.CorrectElevation(wgs84coords.X, wgs84coords.Y, wgs84coords.Z)

	worldMercatorCoords, err := t.coordinateConverter.ConvertCoordinateSrid(
		srid,
		internalCoordinateEpsgCode,
		geometry.Coordinate{
			X: coordinate.X,
			Y: coordinate.Y,
			Z: z,
		},
	)
	x := worldMercatorCoords.X
	y := worldMercatorCoords.Y
	z = worldMercatorCoords.Z
	if !t.offsetInit {
		t.offX = x
		t.offY = y
		t.offZ = z
		t.offsetInit = true
	}
	x -= t.offX
	y -= t.offY
	z -= t.offZ

	if err != nil {
		log.Fatal(err)
	}

	return data.NewPoint(float32(x), float32(y), float32(z), r, g, b, intensity, classification)
}

func (t *GridTree) init() {
	box := t.GetBounds()
	node := NewGridNode(nil, geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), t.maxCellSize, t.minCellSize)
	t.rootNode = node
	t.InitializeLoader()
}

func (t *GridTree) launchParallelPointLoaders(waitGroup *sync.WaitGroup) {
	N := runtime.NumCPU()

	for i := 0; i < N; i++ {
		waitGroup.Add(1)
		go t.launchPointLoader(waitGroup)
	}
}

func (t *GridTree) launchPointLoader(waitGroup *sync.WaitGroup) {
	for {
		val, shouldContinue := t.Loader.GetNext()
		if val != nil {
			t.rootNode.AddDataPoint(val)
		}
		if !shouldContinue {
			break
		}
	}
	waitGroup.Done()
}

func (t *GridTree) GetOffset() (x, y, z float64) {
	return t.offX, t.offY, t.offZ
}
