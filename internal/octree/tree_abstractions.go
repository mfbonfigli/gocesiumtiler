package octree

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/las"
)

type ITree interface {
	Build(las.LasReader) error
	GetRootNode() INode
	IsBuilt() bool
	// Adds a Point to the Tree
	AddPoint(coordinate *geometry.Coordinate, r uint8, g uint8, b uint8, intensity uint8, classification uint8, srid int)
}

type INode interface {
	AddDataPoint(element *data.Point)
	GetInternalSrid() int
	IsRoot() bool
	GetBoundingBoxRegion(converter converters.CoordinateConverter) (*geometry.BoundingBox, error)
	GetChildren() [8]INode
	GetPoints() []*data.Point
	IsEmpty() bool
	NumberOfPoints() int32
	IsLeaf() bool
	IsInitialized() bool
	ComputeGeometricError() float64
	GetParent() INode
	GetBoundingBox() *geometry.BoundingBox
}
