package random_trees

import "C"
import (
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/coordinate/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"math"
	"strings"
	"sync"
	"sync/atomic"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not. Each Node can contain
// up to eight children OctNodes
type octNode struct {
	parent              octree.INode
	boundingBox         *geometry.BoundingBox
	children            [8]octree.INode
	points              []*data.Point
	depth               uint8
	totalNumberOfPoints int64
	numberOfPoints      int32
	opts                *tiler.TilerOptions
	leaf                bool
	initialized         bool
	sync.RWMutex
}

// Instantiates a new octNode
func NewOctNode(boundingBox *geometry.BoundingBox, opts *tiler.TilerOptions, depth uint8, parent octree.INode) octree.INode {
	octNode := octNode{
		parent:              parent,
		boundingBox:         boundingBox,
		depth:               depth,
		opts:                opts,
		totalNumberOfPoints: 0,
		numberOfPoints:      0,
		leaf:                true,
		initialized:         false,
	}

	return &octNode
}

// Adds a Point to the octNode eventually propagating it to the octNode relevant children
func (node *octNode) AddDataPoint(element *data.Point) {
	if atomic.LoadInt32(&node.numberOfPoints) == 0 {
		node.Lock()
		for i := uint8(0); i < 8; i++ {
			if node.children[i] == nil {
				node.children[i] = NewOctNode(getOctantBoundingBox(&i, node.boundingBox), node.opts, node.depth+1, node)
			}
		}
		node.initialized = true
		node.Unlock()
	}
	if atomic.LoadInt32(&node.numberOfPoints) < node.opts.MaxNumPointsPerNode {
		node.Lock()
		node.points = append(node.points, element)
		atomic.AddInt32(&node.numberOfPoints, 1)
		node.Unlock()
	} else {
		node.children[getOctantFromElement(element, node.boundingBox)].AddDataPoint(element)
		if node.leaf {
			node.Lock()
			node.leaf = false
			node.Unlock()
		}
	}
	atomic.AddInt64(&node.totalNumberOfPoints, 1)
}

func (node *octNode) GetParent() octree.INode {
	return node.parent
}

func (node *octNode) GetBoundingBox() *geometry.BoundingBox {
	return node.boundingBox
}

func (node *octNode) GetChildren() [8]octree.INode {
	return node.children
}

func (node *octNode) GetPoints() []*data.Point {
	return node.points
}

func (node *octNode) GetDepth() uint8 {
	return node.depth
}

func (node *octNode) TotalNumberOfPoints() int64 {
	return node.totalNumberOfPoints
}

func (node *octNode) NumberOfPoints() int32 {
	return node.numberOfPoints
}

func (node *octNode) IsLeaf() bool {
	return node.leaf
}

func (node *octNode) IsInitialized() bool {
	return node.initialized
}

// Prints the summary of the node contents in the console
func (node *octNode) PrintStructure() {
	fmt.Println(strings.Repeat(" ", int(node.depth)-1)+"-", "element no:", node.numberOfPoints, "leaf:", node.leaf)
	for _, e := range node.children {
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

// Computes the geometric error for the given octNode
func (node *octNode) ComputeGeometricError() float64 {
	if node.isRootNodeAndLeafNode() {
		return node.estimateErrorAsBoundingBoxDiagonal()
	}

	return node.estimateErrorAsDensityDifference()
}

func (node *octNode) estimateErrorAsBoundingBoxDiagonal() float64 {
	region, _ := proj4_coordinate_converter.NewProj4CoordinateConverter().Convert2DBoundingboxToWGS84Region(node.GetBoundingBox(), node.opts.Srid)
	var latA = region[1]
	var latB = region[3]
	var lngA = region[0]
	var lngB = region[2]
	latA = region[1]
	return 6371000 * math.Acos(math.Cos(latA)*math.Cos(latB)*math.Cos(lngB-lngA)+math.Sin(latA)*math.Sin(latB))
}

func (node *octNode) estimateErrorAsDensityDifference() float64 {
	volume := node.GetBoundingBox().GetWGS84Volume()
	totalRenderedPoints := int64(node.NumberOfPoints())
	parent := node.GetParent()
	for parent != nil {
		for _, e := range parent.GetPoints() {
			if canBoundingBoxContainElement(e, node.GetBoundingBox()) {
				totalRenderedPoints++
			}
		}
		parent = parent.GetParent()
	}
	densityWithAllPoints := math.Pow(volume/float64(totalRenderedPoints+node.TotalNumberOfPoints()-int64(node.NumberOfPoints())), 0.333)
	densityWithOnlyThisTile := math.Pow(volume/float64(totalRenderedPoints), 0.333)

	return densityWithOnlyThisTile - densityWithAllPoints
}
// Checks if the bounding box contains the given element
func canBoundingBoxContainElement(e *data.Point, bbox *geometry.BoundingBox) bool {
	return (e.X >= bbox.Xmin && e.X <= bbox.Xmax) &&
		(e.Y >= bbox.Ymin && e.Y <= bbox.Ymax) &&
		(e.Z >= bbox.Zmin && e.Z <= bbox.Zmax)
}

func (node *octNode) isRootNodeAndLeafNode() bool {
	return node.GetParent() == nil && node.IsLeaf()
}