package octree

import "C"
import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// Contains data necessary to build the octree
type OctNode struct {
	Parent              *OctNode
	BoundingBox         *BoundingBox
	Children            [8]*OctNode
	Items               []*OctElement
	Depth               uint8
	GlobalChildrenCount int64
	LocalChildrenCount  int32
	Opts                *TilerOptions
	IsLeaf              bool
	Initialized         bool
	sync.RWMutex
}

// Instantiates a new OctNode properly initializing the data
func NewOctNode(boundingBox *BoundingBox, opts *TilerOptions, depth uint8, parent *OctNode) *OctNode {
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

// Adds an OctElement to the OctNode eventually propagating it to the OctNode relevant children
func (octNode *OctNode) AddOctElement(element *OctElement) {
	if atomic.LoadInt32(&octNode.LocalChildrenCount) == 0 {
		octNode.Lock()
		for i := uint8(0); i < 8; i++ {
			if octNode.Children[i] == nil {
				octNode.Children[i] = NewOctNode(octNode.BoundingBox.getOctantBoundingBox(&i), octNode.Opts, octNode.Depth+1, octNode)
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
		//octNode.getSubOctNodeContainingElement(element).AddOctElement(element)
		octNode.Children[octNode.BoundingBox.getOctantFromElement(element)].AddOctElement(element)
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
