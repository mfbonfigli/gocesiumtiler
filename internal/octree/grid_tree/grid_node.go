package grid_tree

import "C"
import (
	"math"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
)

// Models a node of the octree, which can either be a leaf (a node without children nodes) or not.
// Each Node can contain up to eight children nodes. The node uses a grid algorithm to decide which points to store.
// It divides its bounding box in gridCells and only stores points retained by these cells, propagating the ones rejected
// by the cells to its children which will have smaller cells.
type GridNode struct {
	parent      octree.INode
	boundingBox *geometry.BoundingBox
	children    [8]octree.INode
	cells       map[gridIndex]*gridCell
	cellSize    float64
	minCellSize float64
	sync.RWMutex
}

// Instantiates a new GridNode
func NewGridNode(parent octree.INode, boundingBox *geometry.BoundingBox, maxCellSize float64, minCellSize float64) octree.INode {
	node := GridNode{
		parent:      parent,                           // the parent node
		boundingBox: boundingBox,                      // bounding box of the node
		cellSize:    maxCellSize,                      // max size setting to use for gridCells
		minCellSize: minCellSize,                      // min size setting to use for gridCells
		cells:       make(map[gridIndex]*gridCell, 0), // gridCells that subdivide this node bounding box
	}

	return &node
}

// Adds a Point to the GridNode and propagates the point eventually pushed out to the appropriate children
func (n *GridNode) AddDataPoint(point *data.Point) {
	if point == nil {
		return
	}

	pushedOutPoint := n.pushPointToCell(point)

	if pushedOutPoint != nil {
		// a point needs to go one level deeper
		n.initializeChildrenIfNeeded()
		n.addPointToChildren(pushedOutPoint)
	}
}

func (n *GridNode) GetInternalSrid() int {
	return internalCoordinateEpsgCode
}

func (n *GridNode) GetBoundingBoxRegion(converter converters.CoordinateConverter, offX, offY, offZ float64) (*geometry.BoundingBox, error) {
	reg, err := converter.Convert2DBoundingboxToWGS84Region(n.boundingBox, n.GetInternalSrid(), offX, offY, offZ)

	if err != nil {
		return nil, err
	}

	return reg, nil
}

func (n *GridNode) GetBoundingBox(offX, offY, offZ float64) *geometry.BoundingBox {
	return n.boundingBox.FromOffset(offX, offY, offZ)
}

func (n *GridNode) GetChildren() [8]octree.INode {
	n.RLock()
	defer n.RUnlock()
	return n.children
}

func (n *GridNode) GetPoints() []*data.Point {
	n.RLock()
	defer n.RUnlock()
	var pts []*data.Point
	for _, gc := range n.cells {
		pts = append(pts, gc.points...)
	}
	return pts
}

func (n *GridNode) IsEmpty() bool {
	n.RLock()
	defer n.RUnlock()
	for _, c := range n.children {
		if c != nil {
			return false
		}
	}

	for _, cell := range n.cells {
		if len(cell.points) > 0 {
			return false
		}
	}

	return true
}

func (n *GridNode) NumberOfPoints() int32 {
	n.RLock()
	defer n.RUnlock()
	var num int32
	for _, cell := range n.cells {
		num += int32(len(cell.points))
	}
	return num
}

func (n *GridNode) IsLeaf() bool {
	n.RLock()
	defer n.RUnlock()
	for _, c := range n.children {
		if c != nil {
			return false
		}
	}
	return true
}

func (n *GridNode) IsRoot() bool {
	return n.parent == nil
}

// Computes the geometric error for the given GridNode
func (n *GridNode) ComputeGeometricError(offX, offY, offZ float64) float64 {
	n.RLock()
	defer n.RUnlock()
	if n.IsRoot() {
		var w = math.Abs(n.boundingBox.Xmax - n.boundingBox.Xmin)
		var l = math.Abs(n.boundingBox.Ymax - n.boundingBox.Ymin)
		var h = math.Abs(n.boundingBox.Zmax - n.boundingBox.Zmin)
		return math.Sqrt(w*w + l*l + h*h)
	}
	// geometric error is estimated as the maximum possible distance between two points lying in the cell
	return n.cellSize * math.Sqrt(3) * 2
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

// not needed, points are computed dynamically when required
func (n *GridNode) BuildPoints() {
	return
}

func (n *GridNode) GetParent() octree.INode {
	return n.parent
}

// gets the grid cell where the given point falls into, eventually creating it if it does not exist
func (n *GridNode) getPointGridCell(index gridIndex) *gridCell {
	n.RLock()
	cell := n.cells[index]
	n.RUnlock()

	if cell == nil {
		return n.initializeGridCell(&index)
	}

	return cell
}

// returns the index of the cell where the input point is falling in
func (n *GridNode) getPointGridCellIndex(point *data.Point) *gridIndex {
	return &gridIndex{
		getDimensionIndex(point.X, n.cellSize),
		getDimensionIndex(point.Y, n.cellSize),
		getDimensionIndex(point.Z, n.cellSize),
	}
}

func (n *GridNode) initializeGridCell(index *gridIndex) *gridCell {
	n.Lock()

	out := n.cells[*index]
	if out == nil {
		out = &gridCell{}
		n.cells[*index] = out
	}

	n.Unlock()

	return out
}

// pushes a point to its gridcell and returns the point eventually pushed out
func (n *GridNode) pushPointToCell(point *data.Point) *data.Point {
	index := *n.getPointGridCellIndex(point)

	return n.getPointGridCell(index).pushPoint(point, n.cellSize, n.minCellSize, index.x, index.y, index.z)
}

// add a point to the node children and clears the leaf flag from this node
func (n *GridNode) addPointToChildren(point *data.Point) {
	n.RLock()
	defer n.RUnlock()
	n.children[getOctantFromElement(point, n.boundingBox)].AddDataPoint(point)
}

func (n *GridNode) IsInitialized() bool {
	return true
}

// initializes the children to new empty nodes
func (n *GridNode) initializeChildrenIfNeeded() {
	n.RLock()
	if n.children[0] != nil {
		n.RUnlock()
		return
	}
	n.RUnlock()
	n.Lock()
	for i := uint8(0); i < 8; i++ {
		if n.children[i] == nil {
			n.children[i] = NewGridNode(n, getOctantBoundingBox(&i, n.boundingBox), n.cellSize/2.0, n.minCellSize)
		}
	}
	n.Unlock()
}

// Returns a bounding box from the given box and the given octant index
func getOctantBoundingBox(octant *uint8, bbox *geometry.BoundingBox) *geometry.BoundingBox {
	return geometry.NewBoundingBoxFromParent(bbox, octant)
}
