package tree

import (
	"context"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/conv/coor"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/mutator"
)

type MockNode struct {
	Bounds                    geom.BoundingBox
	ChildNodes                [8]Node
	Pts                       geom.PointList
	TotalNumPts               int
	Root                      bool
	Leaf                      bool
	GeomError                 float64
	CenterX, CenterY, CenterZ float64
	// invocation params
	Las         las.LasReader
	ConvFactory coor.ConverterFactory
	Mut         mutator.Mutator
	Ctx         context.Context
	LoadCalled  bool
	BuildCalled bool
	Transform   *model.Transform
}

func (n *MockNode) ToParentCRS() *model.Transform {
	return n.Transform
}
func (n *MockNode) BoundingBox() geom.BoundingBox {
	return n.Bounds
}
func (n *MockNode) ChildrenAt(i uint8) Node {
	if val := n.ChildNodes[i]; val != nil {
		return val
	}
	return nil
}
func (n *MockNode) Dispose() error {
	return nil
}
func (n *MockNode) Points() geom.PointList {
	return n.Pts
}
func (n *MockNode) TotalNumberOfPoints() int {
	return n.TotalNumPts
}
func (n *MockNode) NumberOfPoints() int {
	return n.Pts.Len()
}
func (n *MockNode) IsRoot() bool {
	return n.Root
}
func (n *MockNode) IsLeaf() bool {
	return n.Leaf
}
func (n *MockNode) GeometricError() float64 {
	return n.GeomError
}
func (n *MockNode) Build() error {
	n.BuildCalled = true
	return nil
}
func (n *MockNode) RootNode() Node {
	return n
}
func (n *MockNode) Load(l las.LasReader, c coor.ConverterFactory, m mutator.Mutator, ctx context.Context) error {
	n.LoadCalled = true
	n.Ctx = ctx
	n.Las = l
	n.ConvFactory = c
	n.Mut = m
	return nil
}
