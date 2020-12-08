package octree

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
)

type ITree interface {
	Build() error
	GetRootNode() INode
	IsBuilt() bool
	PrintStructure()
	AddPoint(e *data.Point)
}

type INode interface {
	AddDataPoint(element *data.Point)
	GetParent() INode
	GetBoundingBox() *geometry.BoundingBox
	GetChildren() [8]INode
	GetPoints() []*data.Point
	GetDepth() uint8
	GetGlobalChildrenCount() int64
	GetLocalChildrenCount() int32
	IsLeaf() bool
	IsInitialized() bool
	PrintStructure()
}
