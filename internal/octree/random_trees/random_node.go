package random_trees

import "C"
import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/coordinate/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not. Each Node can contain
// up to eight children nodes
type RandomNode struct {
	parent              octree.INode
	boundingBox         *geometry.BoundingBox
	children            [8]octree.INode
	points              []*data.Point
	internalSrid        int
	totalNumberOfPoints int64
	numberOfPoints      int32
	tilerOptions        *tiler.TilerOptions
	leaf                bool
	initialized         bool
	sync.RWMutex
}

// Instantiates a new RandomNode
func NewRandomNode(boundingBox *geometry.BoundingBox, opts *tiler.TilerOptions, parent octree.INode) octree.INode {
	node := RandomNode{
		parent:              parent,
		boundingBox:         boundingBox,
		internalSrid:        4326,
		tilerOptions:        opts,
		totalNumberOfPoints: 0,
		numberOfPoints:      0,
		leaf:                true,
		initialized:         false,
	}

	return &node
}

// Adds a Point to the RandomNode eventually propagating it to the RandomNode relevant children
func (n *RandomNode) AddDataPoint(element *data.Point) {
	if atomic.LoadInt32(&n.numberOfPoints) == 0 {
		n.Lock()
		for i := uint8(0); i < 8; i++ {
			if n.children[i] == nil {
				n.children[i] = NewRandomNode(getOctantBoundingBox(&i, n.boundingBox), n.tilerOptions, n)
			}
		}
		n.initialized = true
		n.Unlock()
	}
	if atomic.LoadInt32(&n.numberOfPoints) < n.tilerOptions.MaxNumPointsPerNode {
		n.Lock()
		n.points = append(n.points, element)
		atomic.AddInt32(&n.numberOfPoints, 1)
		n.Unlock()
	} else {
		n.children[getOctantFromElement(element, n.boundingBox)].AddDataPoint(element)
		if n.leaf {
			n.Lock()
			n.leaf = false
			n.Unlock()
		}
	}
	atomic.AddInt64(&n.totalNumberOfPoints, 1)
}

func (n *RandomNode) GetParent() octree.INode {
	return n.parent
}

func (n *RandomNode) GetInternalSrid() int {
	return n.internalSrid
}

func (n *RandomNode) GetBoundingBoxRegion(converter converters.CoordinateConverter) (*geometry.BoundingBox, error) {
	reg, err := converter.Convert2DBoundingboxToWGS84Region(n.boundingBox, n.GetInternalSrid())

	if err != nil {
		return nil, err
	}

	return reg, nil
}

func (n *RandomNode) GetBoundingBox() *geometry.BoundingBox {
	return n.boundingBox
}

func (n *RandomNode) GetChildren() [8]octree.INode {
	return n.children
}

func (n *RandomNode) GetPoints() []*data.Point {
	return n.points
}

func (n *RandomNode) IsEmpty() bool {
	return n.totalNumberOfPoints == 0
}

func (n *RandomNode) NumberOfPoints() int32 {
	return n.numberOfPoints
}

func (n *RandomNode) IsLeaf() bool {
	return n.leaf
}

func (n *RandomNode) IsRoot() bool {
	return n.parent == nil
}

func (n *RandomNode) IsInitialized() bool {
	return n.initialized
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

// Computes the geometric error for the given RandomNode
func (n *RandomNode) ComputeGeometricError() float64 {
	if n.isRootNodeAndLeafNode() {
		return n.estimateErrorAsBoundingBoxDiagonal()
	}

	return n.estimateErrorAsDensityDifference()
}

func (n *RandomNode) estimateErrorAsBoundingBoxDiagonal() float64 {
	regionBox, _ := proj4_coordinate_converter.NewProj4CoordinateConverter().Convert2DBoundingboxToWGS84Region(n.boundingBox, n.GetInternalSrid())
	region := regionBox.GetAsArray()
	var latA = region[1]
	var latB = region[3]
	var lngA = region[0]
	var lngB = region[2]
	latA = region[1]
	return 6371000 * math.Acos(math.Cos(latA)*math.Cos(latB)*math.Cos(lngB-lngA)+math.Sin(latA)*math.Sin(latB))
}

func (n *RandomNode) estimateErrorAsDensityDifference() float64 {
	volume := n.boundingBox.GetWGS84Volume()
	totalRenderedPoints := int64(n.NumberOfPoints())
	parent := n.GetParent()
	for parent != nil {
		for _, e := range parent.GetPoints() {
			if canBoundingBoxContainElement(e, n.boundingBox) {
				totalRenderedPoints++
			}
		}
		parent = parent.(*RandomNode).parent
	}
	densityWithAllPoints := math.Pow(volume/float64(totalRenderedPoints+n.totalNumberOfPoints-int64(n.NumberOfPoints())), 0.333)
	densityWithOnlyThisTile := math.Pow(volume/float64(totalRenderedPoints), 0.333)

	return densityWithOnlyThisTile - densityWithAllPoints
}

// Checks if the bounding box contains the given element
func canBoundingBoxContainElement(e *data.Point, bbox *geometry.BoundingBox) bool {
	return (e.X >= bbox.Xmin && e.X <= bbox.Xmax) &&
		(e.Y >= bbox.Ymin && e.Y <= bbox.Ymax) &&
		(e.Z >= bbox.Zmin && e.Z <= bbox.Zmax)
}

func (n *RandomNode) isRootNodeAndLeafNode() bool {
	return n.GetParent() == nil && n.IsLeaf()
}
