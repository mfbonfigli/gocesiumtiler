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
	"runtime"
	"sync"
)

// Represents an RandomTree of points and contains all information needed
// to propagate points in the tree
type RandomTree struct {
	rootNode            octree.INode
	built               bool
	opts                *tiler.TilerOptions
	coordinateConverter converters.CoordinateConverter
	elevationCorrector  converters.ElevationCorrector
	point_loader.Loader
}

// Builds an empty RandomTree initializing its properties to the correct defaults
func NewRandomTree(opts *tiler.TilerOptions, coordinateConverter converters.CoordinateConverter, elevationCorrector converters.ElevationCorrector) octree.ITree {
	return &RandomTree{
		built:               false,
		opts:                opts,
		Loader:              point_loader.NewRandomLoader(),
		coordinateConverter: coordinateConverter,
		elevationCorrector:  elevationCorrector,
	}
}

func NewBoxedRandomTree(opts *tiler.TilerOptions, coordinateConverter converters.CoordinateConverter, elevationCorrector converters.ElevationCorrector) octree.ITree {
	return &RandomTree{
		built:               false,
		opts:                opts,
		Loader:              point_loader.NewRandomBoxLoader(),
		coordinateConverter: coordinateConverter,
		elevationCorrector:  elevationCorrector,
	}
}

// Builds the hierarchical tree structure propagating the added items according to the TilerOptions provided
// during initialization
func (t *RandomTree) Build() error {
	if t.built {
		return errors.New("octree already built")
	}

	t.init()

	var wg sync.WaitGroup
	t.launchParallelPointLoaders(&wg)
	wg.Wait()

	t.built = true

	return nil
}

func (t *RandomTree) init() {
	box := t.GetBounds()
	node := NewRandomNode(geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), t.opts, nil)
	t.rootNode = node
	t.InitializeLoader()
}

func (t *RandomTree) launchParallelPointLoaders(waitGroup *sync.WaitGroup) {
	N := runtime.NumCPU()

	for i := 0; i < N; i++ {
		waitGroup.Add(1)
		go t.launchPointLoader(waitGroup)
	}
}

func (t *RandomTree) launchPointLoader(waitGroup *sync.WaitGroup) {
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

func (t *RandomTree) GetRootNode() octree.INode {
	return t.rootNode
}

func (t *RandomTree) IsBuilt() bool {
	return t.built
}

func (t *RandomTree) AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) {
	t.Loader.AddPoint(t.getPointFromRawData(coordinate, r, g, b, intensity, classification, srid))
}

func (t *RandomTree) getPointFromRawData(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int) *data.Point {
	tr, err := t.coordinateConverter.ConvertCoordinateSrid(srid, 4326, *coordinate)
	if err != nil {
		log.Fatal(err)
	}

	return data.NewPoint(tr.X, tr.Y, t.elevationCorrector.CorrectElevation(tr.X, tr.Y, tr.Z), r, g, b, intensity, classification)
}
