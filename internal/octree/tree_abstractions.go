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
	// Adds a Point to the Tree
	AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int)
}

type INode interface {
	AddDataPoint(element *data.Point)
	GetInternalSrid() int
	GetParent() INode
	GetBoundingBox() *geometry.BoundingBox
	GetChildren() [8]INode
	GetPoints() []*data.Point
	GetDepth() uint8
	TotalNumberOfPoints() int64
	NumberOfPoints() int32
	IsLeaf() bool
	IsInitialized() bool
	PrintStructure()
	ComputeGeometricError() float64
}
