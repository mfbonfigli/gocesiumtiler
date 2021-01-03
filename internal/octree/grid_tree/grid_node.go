package grid_tree

import "C"
import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"math"
	"sync"
	"sync/atomic"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not.
// Each Node can contain up to eight children nodes. The node uses a grid algorithm to decide which points to store.
// It divides its bounding box in gridCells and only stores points retained by these cells, propagating the ones rejected
// by the cells to its children which will have smaller cells.
type GridNode struct {
	root                bool
	boundingBox         *geometry.BoundingBox
	children            [8]octree.INode
	cells               map[gridIndex]*gridCell
	points              []*data.Point
	cellSize            float64
	minCellSize         float64
	totalNumberOfPoints int64
	numberOfPoints      int32
	leaf                int32
	initialized         bool
	sync.RWMutex
}

// Instantiates a new GridNode
func NewGridNode(boundingBox *geometry.BoundingBox, maxCellSize float64, minCellSize float64, root bool) octree.INode {
	node := GridNode{
		root:                root,                             // if the node is the tree root
		boundingBox:         boundingBox,                      // bounding box of the node
		cellSize:            maxCellSize,                      // max size setting to use for gridCells
		minCellSize:         minCellSize,                      // min size setting to use for gridCells
		points:              make([]*data.Point, 0),           // slice keeping references to points stored in the gridCells
		cells:               make(map[gridIndex]*gridCell, 0), // gridCells that subdivide this node bounding box
		totalNumberOfPoints: 0,                                // total number of points stored in this node and its children
		numberOfPoints:      0,                                // number of points stored in this node (children excluded)
		leaf:                1,                                // 1 if is a leaf, 0 otherwise
		initialized:         false,                            // flag to see if the node has been initialized
	}

	return &node
}

// Adds a Point to the GridNode and propagates the point eventually pushed out to the appropriate children
func (node *GridNode) AddDataPoint(point *data.Point) {
	if point == nil {
		return
	}

	if node.isEmpty() {
		node.initializeChildren()
	}

	pushedOutPoint := node.pushPointToCell(point)

	if pushedOutPoint != nil {
		node.addPointToChildren(pushedOutPoint)
	} else {
		// if no point was rejected then the number of points stored is increased by 1
		atomic.AddInt32(&node.numberOfPoints, 1)
	}

	// in any case the total number of points stored by the node or its children increases by one
	atomic.AddInt64(&node.totalNumberOfPoints, 1)
}

func (node *GridNode) GetInternalSrid() int {
	return internalCoordinateEpsgCode
}

func (node *GridNode) GetBoundingBoxRegion(converter converters.CoordinateConverter) ([]float64, error) {
	reg, err := converter.Convert2DBoundingboxToWGS84Region(node.boundingBox, node.GetInternalSrid())

	if err != nil {
		return nil, err
	}

	return reg, nil
}

func (node *GridNode) GetChildren() [8]octree.INode {
	return node.children
}

func (node *GridNode) GetPoints() []*data.Point {
	return node.points
}

func (node *GridNode) TotalNumberOfPoints() int64 {
	return node.totalNumberOfPoints
}

func (node *GridNode) NumberOfPoints() int32 {
	return node.numberOfPoints
}

func (node *GridNode) IsLeaf() bool {
	return atomic.LoadInt32(&node.leaf) == 1
}

func (node *GridNode) IsInitialized() bool {
	return node.initialized
}

func (node *GridNode) IsRoot() bool {
	return node.root
}

// Computes the geometric error for the given GridNode
func (node *GridNode) ComputeGeometricError() float64 {
	// geometric error is estimated as the maximum possible distance between two points lying in the cell
	return node.cellSize * math.Sqrt(3) * 2
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

// loads the points stored in the grid cells into the slice data structure
// and recursively builds the points of its children.
// sets the slice reference to nil to allow GC to happen as the cells won't be used anymore
func (node *GridNode) buildPoints() {
	var points []*data.Point
	for _, cell := range node.cells {
		points = append(points, cell.points...)
	}
	node.points = points
	node.cells = nil

	for _, child := range node.children {
		if child != nil {
			child.(*GridNode).buildPoints()
		}
	}
}

// gets the grid cell where the given point falls into, eventually creating it if it does not exist
func (node *GridNode) getPointGridCell(point *data.Point) *gridCell {
	index := *node.getPointGridCellIndex(point)

	node.RLock()
	cell := node.cells[index]
	node.RUnlock()

	if cell == nil {
		return node.initializeGridCell(&index)
	}

	return cell
}

// returns the index of the cell where the input point is falling in
func (node *GridNode) getPointGridCellIndex(point *data.Point) *gridIndex {
	return &gridIndex{
		getDimensionIndex(point.X, node.cellSize),
		getDimensionIndex(point.Y, node.cellSize),
		getDimensionIndex(point.Z, node.cellSize),
	}
}

func (node *GridNode) initializeGridCell(index *gridIndex) *gridCell {
	node.Lock()

	out := node.cells[*index]
	if out == nil {
		out = &gridCell{
			index:         *index,
			size:          node.cellSize,
			sizeThreshold: node.minCellSize,
		}
		node.cells[*index] = out
	}

	node.Unlock()

	return out
}

// atomically checks if the node is empty
func (node *GridNode) isEmpty() bool {
	return atomic.LoadInt32(&node.numberOfPoints) == 0
}

// pushes a point to its gridcell and returns the point eventually pushed out
func (node *GridNode) pushPointToCell(point *data.Point) *data.Point {
	return node.getPointGridCell(point).pushPoint(point)
}

// add a point to the node children and clears the leaf flag from this node
func (node *GridNode) addPointToChildren(point *data.Point) {
	node.children[getOctantFromElement(point, node.boundingBox)].AddDataPoint(point)
	node.clearLeafFlag()
}

// sets the leaf flag to 0 atomically
func (node *GridNode) clearLeafFlag() {
	atomic.StoreInt32(&node.leaf, 0)
}

// initializes the children to new empty nodes
func (node *GridNode) initializeChildren() {
	node.Lock()
	for i := uint8(0); i < 8; i++ {
		if node.children[i] == nil {
			node.children[i] = NewGridNode(getOctantBoundingBox(&i, node.boundingBox), node.cellSize/2.0, node.minCellSize, false)
		}
	}
	node.initialized = true
	node.Unlock()
}

// Returns a bounding box from the given box and the given octant index
func getOctantBoundingBox(octant *uint8, bbox *geometry.BoundingBox) *geometry.BoundingBox {
	return geometry.NewBoundingBoxFromParent(bbox, octant)
}
