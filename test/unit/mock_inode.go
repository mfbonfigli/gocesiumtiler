package unit

import "C"
import (
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"sync"
)

// mock implementation of the INode interface
type mockNode struct {
	parent              octree.INode
	boundingBox         *geometry.BoundingBox
	children            [8]octree.INode
	points              []*data.Point
	internalSrid        int
	depth               uint8
	globalChildrenCount int64
	localChildrenCount  int32
	opts                *tiler.TilerOptions
	leaf                bool
	initialized         bool
	geometricError      float64
	sync.RWMutex
}

// Adds a Point to the octNode eventually propagating it to the octNode relevant children
func (mockNode *mockNode) AddDataPoint(element *data.Point) {}

func (mockNode *mockNode) GetParent() octree.INode {
	return mockNode.parent
}

func (mockNode *mockNode) GetBoundingBox() *geometry.BoundingBox {
	return mockNode.boundingBox
}

func (mockNode *mockNode) GetChildren() [8]octree.INode {
	return mockNode.children
}

func (mockNode *mockNode) GetPoints() []*data.Point {
	return mockNode.points
}

func (mockNode *mockNode) GetInternalSrid() int {
	return mockNode.internalSrid
}

func (mockNode *mockNode) GetDepth() uint8 {
	return mockNode.depth
}

func (mockNode *mockNode) TotalNumberOfPoints() int64 {
	return mockNode.globalChildrenCount
}

func (mockNode *mockNode) NumberOfPoints() int32 {
	return mockNode.localChildrenCount
}

func (mockNode *mockNode) IsLeaf() bool {
	return mockNode.leaf
}

func (mockNode *mockNode) IsInitialized() bool {
	return mockNode.initialized
}

// Prints the summary of the node contents in the console
func (mockNode *mockNode) PrintStructure() {}

func (mockNode *mockNode) ComputeGeometricError() float64 {
	return mockNode.geometricError
}
