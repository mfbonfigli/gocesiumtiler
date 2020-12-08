package octree

import (
	"errors"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/point_loader"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"math"
	"runtime"
	"sync"
)

// Represents an octTree of points and contains all information needed
// to propagate points in the tree
type octTree struct {
	itemsToAdd                         []data.Point
	rootNode                           INode
	built                              bool
	minX, maxX, minY, maxY, minZ, maxZ float64
	opts                               *tiler.TilerOptions
	point_loader.Loader
}

// Builds an empty octTree initializing its properties to the correct defaults
func NewRandomTree(opts *tiler.TilerOptions) ITree {
	return &octTree{
		itemsToAdd: make([]data.Point, 0),
		built:      false,
		minX:       math.MaxFloat64,
		minY:       math.MaxFloat64,
		minZ:       math.MaxFloat64,
		maxX:       -1 * math.MaxFloat64,
		maxY:       -1 * math.MaxFloat64,
		maxZ:       -1 * math.MaxFloat64,
		opts:       opts,
		Loader: point_loader.NewRandomLoader(),
	}
}

func NewBoxedRandomTree(opts *tiler.TilerOptions) ITree {
	return &octTree{
		itemsToAdd: make([]data.Point, 0),
		built:      false,
		minX:       math.MaxFloat64,
		minY:       math.MaxFloat64,
		minZ:       math.MaxFloat64,
		maxX:       -1 * math.MaxFloat64,
		maxY:       -1 * math.MaxFloat64,
		maxZ:       -1 * math.MaxFloat64,
		opts:       opts,
		Loader: point_loader.NewRandomBoxLoader(),
	}
}

// Internally update the bounds of the tree.
// TODO: These could be read directly from the LAS file
func (octTree *octTree) recomputeBoundsFromElement(element *data.Point) {
	octTree.minX = math.Min(float64(element.X), octTree.minX)
	octTree.minY = math.Min(float64(element.Y), octTree.minY)
	octTree.minZ = math.Min(float64(element.Z), octTree.minZ)
	octTree.maxX = math.Max(float64(element.X), octTree.maxX)
	octTree.maxY = math.Max(float64(element.Y), octTree.maxY)
	octTree.maxZ = math.Max(float64(element.Z), octTree.maxZ)
}

// Builds the hierarchical tree structure propagating the added items according to the TilerOptions provided
// during initialization
func (octTree *octTree) Build() error {
	if octTree.built {
		return errors.New("octree already built")
	}
	box := octTree.GetBounds()
	octNode := NewOctNode(geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), octTree.opts, 1, nil)
	octTree.rootNode = octNode
	octTree.Initialize()
	var wg sync.WaitGroup
	//wg.Add(len(octTree.itemsToAdd))
	N := runtime.NumCPU()
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(loader point_loader.Loader) {
			for {
				val, shouldContinue := loader.GetNext()
				if val != nil {
					octTree.rootNode.AddDataPoint(val)
				}
				if !shouldContinue {
					break
				}
			}
			wg.Done()
		}(octTree.Loader)
	}
	wg.Wait()
	octTree.itemsToAdd = nil
	octTree.built = true
	return nil
}

// Prints the tree structure
func (octTree *octTree) PrintStructure() {
	if octTree.built {
		octTree.rootNode.PrintStructure()
	}
}

func (octTree *octTree) GetRootNode() INode {
	return octTree.rootNode
}

func (octTree *octTree) IsBuilt() bool {
	return octTree.built
}
