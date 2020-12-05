package octree

import (
	"errors"
	"github.com/mfbonfigli/gocesiumtiler/structs/data"
	"github.com/mfbonfigli/gocesiumtiler/structs/geometry"
	"github.com/mfbonfigli/gocesiumtiler/structs/point_loader"
	"github.com/mfbonfigli/gocesiumtiler/structs/tiler"
	"math"
	"runtime"
	"sync"
)

// Represents an OctTree of Points and contains all information needed
// to propagate points in the tree
type OctTree struct {
	itemsToAdd                         []data.Point
	RootNode                           OctNode
	Built                              bool
	minX, maxX, minY, maxY, minZ, maxZ float64
	Opts                               *tiler.TilerOptions
}

// Builds an empty OctTree initializing its properties to the correct defaults
func NewOctTree(opts *tiler.TilerOptions) *OctTree {
	return &OctTree{
		itemsToAdd: make([]data.Point, 0),
		Built:      false,
		minX:       math.MaxFloat64,
		minY:       math.MaxFloat64,
		minZ:       math.MaxFloat64,
		maxX:       -1 * math.MaxFloat64,
		maxY:       -1 * math.MaxFloat64,
		maxZ:       -1 * math.MaxFloat64,
		Opts:       opts,
	}
}

// Internally update the bounds of the tree.
// TODO: These could be read directly from the LAS file
func (octTree *OctTree) recomputeBoundsFromElement(element *data.Point) {
	octTree.minX = math.Min(float64(element.X), octTree.minX)
	octTree.minY = math.Min(float64(element.Y), octTree.minY)
	octTree.minZ = math.Min(float64(element.Z), octTree.minZ)
	octTree.maxX = math.Max(float64(element.X), octTree.maxX)
	octTree.maxY = math.Max(float64(element.Y), octTree.maxY)
	octTree.maxZ = math.Max(float64(element.Z), octTree.maxZ)
}

// Builds the hierarchical tree structure propagating the added items according to the TilerOptions provided
// during initialization
func (octTree *OctTree) Build(loader point_loader.Loader) error {
	if octTree.Built {
		return errors.New("octree already Built")
	}
	box := loader.GetBounds()
	octNode := NewOctNode(geometry.NewBoundingBox(box[0], box[1], box[2], box[3], box[4], box[5]), octTree.Opts, 1, nil)
	octTree.RootNode = *octNode
	loader.Initialize()
	var wg sync.WaitGroup
	//wg.Add(len(octTree.itemsToAdd))
	N := runtime.NumCPU()
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(loader point_loader.Loader) {
			for {
				val, shouldContinue := loader.GetNext()
				if val != nil {
					octTree.RootNode.AddDataPoint(val)
				}
				if !shouldContinue {
					break
				}
			}
			wg.Done()
		}(loader)
	}
	wg.Wait()
	octTree.itemsToAdd = nil
	octTree.Built = true
	return nil
}

// Prints the tree structure
func (octTree *OctTree) PrintStructure() {
	if octTree.Built {
		octTree.RootNode.PrintStructure()
	}
}
