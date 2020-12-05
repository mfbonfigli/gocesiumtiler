package octree

import "C"
import (
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/structs/data"
	"github.com/mfbonfigli/gocesiumtiler/structs/geometry"
	"github.com/mfbonfigli/gocesiumtiler/structs/tiler"
	"strings"
	"sync"
	"sync/atomic"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not. Each Node can contain
// up to eight children OctNodes
type OctNode struct {
	Parent              *OctNode
	BoundingBox         *geometry.BoundingBox
	Children            [8]*OctNode
	Items               []*data.Point
	Depth               uint8
	GlobalChildrenCount int64
	LocalChildrenCount  int32
	Opts                *tiler.TilerOptions
	IsLeaf              bool
	Initialized         bool
	sync.RWMutex
}

// Instantiates a new OctNode
func NewOctNode(boundingBox *geometry.BoundingBox, opts *tiler.TilerOptions, depth uint8, parent *OctNode) *OctNode {
	octNode := OctNode{
		Parent:              parent,
		BoundingBox:         boundingBox,
		Depth:               depth,
		Opts:                opts,
		GlobalChildrenCount: 0,
		LocalChildrenCount:  0,
		IsLeaf:              true,
		Initialized:         false,
	}

	return &octNode
}

// Adds a Point to the OctNode eventually propagating it to the OctNode relevant children
func (octNode *OctNode) AddDataPoint(element *data.Point) {
	if atomic.LoadInt32(&octNode.LocalChildrenCount) == 0 {
		octNode.Lock()
		for i := uint8(0); i < 8; i++ {
			if octNode.Children[i] == nil {
				octNode.Children[i] = NewOctNode(getOctantBoundingBox(&i, octNode.BoundingBox), octNode.Opts, octNode.Depth+1, octNode)
			}
		}
		octNode.Initialized = true
		octNode.Unlock()
	}
	if atomic.LoadInt32(&octNode.LocalChildrenCount) < octNode.Opts.MaxNumPointsPerNode {
		octNode.Lock()
		octNode.Items = append(octNode.Items, element)
		atomic.AddInt32(&octNode.LocalChildrenCount, 1)
		octNode.Unlock()
	} else {
		octNode.Children[getOctantFromElement(element, octNode.BoundingBox)].AddDataPoint(element)
		if octNode.IsLeaf {
			octNode.Lock()
			octNode.IsLeaf = false
			octNode.Unlock()
		}
	}
	atomic.AddInt64(&octNode.GlobalChildrenCount, 1)
}

// Prints the summary of the node contents in the console
func (octNode *OctNode) PrintStructure() {
	fmt.Println(strings.Repeat(" ", int(octNode.Depth)-1)+"-", "element no:", octNode.LocalChildrenCount, "leaf:", octNode.IsLeaf)
	for _, e := range octNode.Children {
		if e != nil {
			e.PrintStructure()
		}
	}
}




// Returns the index of the octant that contains the given Point within this BoundingBox
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