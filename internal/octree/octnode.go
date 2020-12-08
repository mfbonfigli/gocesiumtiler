package octree

import "C"
import (
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"strings"
	"sync"
	"sync/atomic"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not. Each Node can contain
// up to eight children OctNodes
type octNode struct {
	parent              INode
	boundingBox         *geometry.BoundingBox
	children            [8]INode
	points              []*data.Point
	depth               uint8
	globalChildrenCount int64
	localChildrenCount  int32
	opts                *tiler.TilerOptions
	leaf                bool
	initialized         bool
	sync.RWMutex
}

// Instantiates a new octNode
func NewOctNode(boundingBox *geometry.BoundingBox, opts *tiler.TilerOptions, depth uint8, parent INode) INode {
	octNode := octNode{
		parent:              parent,
		boundingBox:         boundingBox,
		depth:               depth,
		opts:                opts,
		globalChildrenCount: 0,
		localChildrenCount:  0,
		leaf:                true,
		initialized:         false,
	}

	return &octNode
}

// Adds a Point to the octNode eventually propagating it to the octNode relevant children
func (octNode *octNode) AddDataPoint(element *data.Point) {
	if atomic.LoadInt32(&octNode.localChildrenCount) == 0 {
		octNode.Lock()
		for i := uint8(0); i < 8; i++ {
			if octNode.children[i] == nil {
				octNode.children[i] = NewOctNode(getOctantBoundingBox(&i, octNode.boundingBox), octNode.opts, octNode.depth+1, octNode)
			}
		}
		octNode.initialized = true
		octNode.Unlock()
	}
	if atomic.LoadInt32(&octNode.localChildrenCount) < octNode.opts.MaxNumPointsPerNode {
		octNode.Lock()
		octNode.points = append(octNode.points, element)
		atomic.AddInt32(&octNode.localChildrenCount, 1)
		octNode.Unlock()
	} else {
		octNode.children[getOctantFromElement(element, octNode.boundingBox)].AddDataPoint(element)
		if octNode.leaf {
			octNode.Lock()
			octNode.leaf = false
			octNode.Unlock()
		}
	}
	atomic.AddInt64(&octNode.globalChildrenCount, 1)
}

func (octNode *octNode) GetParent() INode {
	return octNode.parent
}

func (octNode *octNode) GetBoundingBox() *geometry.BoundingBox {
	return octNode.boundingBox
}

func (octNode *octNode) GetChildren() [8]INode {
	return octNode.children
}

func (octNode *octNode) GetPoints() []*data.Point {
	return octNode.points
}

func (octNode *octNode) GetDepth() uint8 {
	return octNode.depth
}

func (octNode *octNode) GetGlobalChildrenCount() int64 {
	return octNode.globalChildrenCount
}

func (octNode *octNode) GetLocalChildrenCount() int32 {
	return octNode.localChildrenCount
}

func (octNode *octNode) IsLeaf() bool {
	return octNode.leaf
}

func (octNode *octNode) IsInitialized() bool {
	return octNode.initialized
}

// Prints the summary of the node contents in the console
func (octNode *octNode) PrintStructure() {
	fmt.Println(strings.Repeat(" ", int(octNode.depth)-1)+"-", "element no:", octNode.localChildrenCount, "leaf:", octNode.leaf)
	for _, e := range octNode.children {
		if e != nil {
			e.PrintStructure()
		}
	}
}

// Returns the index of the octant that contains the given Point within this boundingBox
func getOctantFromElement(element *data.Point, bbox *geometry.BoundingBox) uint8 {
	var result uint8 = 0
	if float64(element.X) > bbox.Xmid {
		result += 1
	}
	if float64(element.Y) > bbox.Ymid {
		result += 2
	}
	if float64(element.Z) > bbox.Zmid {
		result += 4
	}
	return result
}

// Returns a bounding box from the given box and the given octant index
func getOctantBoundingBox(octant *uint8, bbox *geometry.BoundingBox) *geometry.BoundingBox {
	return geometry.NewBoundingBoxFromParent(bbox, octant)
}
