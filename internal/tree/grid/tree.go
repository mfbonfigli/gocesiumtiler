package grid

import (
	"context"
	"math"
	"math/rand"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/conv/coor"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/mutator"
)

// Node implements both the Tree and Node interfaces. The points of the point cloud
// are stored according to a local CRS, which can be transformed back to the EPSG 4978 CRS via a transform
// object stored in the tree root node. The characteristic of this Node implementation is that
// sampling is performed by determining a virtual "grid" at each level of detail, which helps achieve uniform
// spatial sampling. The grid has a spacing in meters.
//
// The build operation will:
//   - partition the space according to the grid
//   - given a space partition, retain the point that belongs to the partition and that is closest to its center
//     unless the maximum depth of the tree is reached, in which case all points are retained.
//   - store all other points no retained to be used to build the children
//
// The tree is "lazy". It never builds the children until they are queried.
type Node struct {
	// pts is a linked list of points in local coordinates belonging to this Node
	pts *geom.LinkedPoint

	// childrenPts that temporarily stores the points that should
	// fall into the 8 children octants before these are built
	childrenPts [8]*geom.LinkedPoint

	// children contains pointers to the child nodes of the tree
	children [8]tree.Node

	// childrenBuilt is true if the children have been properly built
	childrenBuilt bool

	// bounds stores the bounding box for the current node, in local coordinates
	bounds geom.BoundingBox

	// gridSize stores the sampling interval of points used to sample
	// the cloud at this node tree depth
	gridSize float64

	// built is true if the Build method was called on the Node
	built bool

	// maxDepth is the maximum depth the tree can reach
	maxDepth int

	// depth is the actual depth of the node
	depth int

	// numPoints stores the number of points directly contained in the Node
	// without including points in the children
	numPoints int

	// totalNumPoints stores the total number of points stored in the node
	// or its children
	totalNumPoints int

	// loadWorkersNumber is the number of parallel workers to use to load
	// points in the node
	loadWorkersNumber int

	// minPointsPerChildren is the minimum numbr of points a children can contain,
	// if less its points will be rolled up to the parent
	minPointsPerChildren int

	// maxPointsPerTile is the maximum number of points a tile can contain.
	// If a tile contains more points, a subsampling strategy is applied.
	maxPointsPerTile int

	// localToGlobal is a pointer to the Transform matrix that convers this node coordinates
	// into the parent coordinates. If nil the identity trasform is implied. For the Root node
	// of a tree this localToGlobal convers the local coordinates into the EPSG 4978 CRS.
	localToGlobal *model.Transform

	sync.Mutex
}

// NewTree returns a new tree with default settings
func NewTree(opts ...func(*Node)) *Node {
	t := &Node{
		built:                false,
		maxDepth:             10,
		depth:                0,
		childrenBuilt:        false,
		gridSize:             1,
		loadWorkersNumber:    1,
		minPointsPerChildren: 2000,
		maxPointsPerTile:     160000, // 0 means no limit
		localToGlobal:        nil,
	}
	for _, optFn := range opts {
		optFn(t)
	}
	return t
}

// WithGridSize sets the sampling interval for the outermost tree node. The interval
// is halved at every level
func WithGridSize(size float64) func(t *Node) {
	return func(t *Node) {
		t.gridSize = size
	}
}

// WithMaxDepth sets the max number of levels of the tree
func WithMaxDepth(depth int) func(t *Node) {
	return func(t *Node) {
		t.maxDepth = depth
	}
}

// WithLoadWorkersNumber sets the number of parallel goroutines to use to read from the las file
func WithLoadWorkersNumber(num int) func(t *Node) {
	return func(t *Node) {
		t.loadWorkersNumber = num
	}
}

// WithMinPointsPerChildren sets the minimum number of points a children node should contain,
// if that is not possible the children points will be rolled up to its parent
func WithMinPointsPerChildren(num int) func(t *Node) {
	return func(t *Node) {
		t.minPointsPerChildren = num
	}
}

// WithMaxPointsPerTile sets the maximum number of points a tile can contain.
// If a tile contains more points, a subsampling strategy is applied.
func WithMaxPointsPerTile(num int) func(t *Node) {
	return func(t *Node) {
		t.maxPointsPerTile = num
	}
}

// Loads points into the tree from the given las converting them into local coordinates and setting the node transform correctly
func (t *Node) Load(reader las.LasReader, coorConv coor.ConverterFactory, mut mutator.Mutator, ctx context.Context) error {
	return t.loadPoints(reader, coorConv, mut, ctx)
}

func (t *Node) RootNode() tree.Node {
	if t.depth == 0 {
		return tree.Node(t)
	}
	return nil
}

func (t *Node) Build() error {
	if t.depth >= t.maxDepth {
		// reached maxDepth, swallow in all points
		current := t.pts
		// traverse just to update the internal counters
		for current != nil {
			t.totalNumPoints++
			t.numPoints++
			current = current.Next
		}
		// max depth, no further subdivision possible, mark as built and return
		t.built = true
		return nil
	}

	// First pass of grid sampling
	winners, losers, numWinners, totalPointsProcessed := t.sampleGrid(t.pts, t.gridSize)
	t.totalNumPoints = totalPointsProcessed

	// Iterative re-sampling if needed
	if t.maxPointsPerTile > 0 && numWinners > t.maxPointsPerTile {
		currentGridSize := t.gridSize
		// Safeguard: do not exceed 2x the initial grid size (which is the parent's grid size)
		maxGridSize := t.gridSize * 2

		for numWinners > t.maxPointsPerTile && currentGridSize < maxGridSize {
			// Estimate new grid size
			scalingFactor := math.Pow(float64(numWinners)/float64(t.maxPointsPerTile), 1.0/3.0) * 1.1
			newGridSize := currentGridSize * scalingFactor

			if newGridSize > maxGridSize {
				newGridSize = maxGridSize
			}

			// Re-sample with the new grid size.
			// Note: we sample from the 'winners' of the previous iteration.
			newWinners, newLosers, newNumWinners, _ := t.sampleGrid(winners, newGridSize)
			numWinners = newNumWinners
			currentGridSize = newGridSize

			// Add the new losers to the main losers list
			if newLosers != nil {
				// Find the end of the main losers list
				end := losers
				if end != nil {
					for end.Next != nil {
						end = end.Next
					}
					end.Next = newLosers
				} else {
					losers = newLosers
				}
			}
			winners = newWinners
		}

		// Fallback: if still too many points, do random sampling
		if numWinners > t.maxPointsPerTile {
			// Convert linked list to slice for shuffling
			winnerSlice := make([]*geom.LinkedPoint, 0, numWinners)
			cur := winners
			for cur != nil {
				winnerSlice = append(winnerSlice, cur)
				cur = cur.Next
			}

			// Shuffle and split
			rand.Shuffle(len(winnerSlice), func(i, j int) {
				winnerSlice[i], winnerSlice[j] = winnerSlice[j], winnerSlice[i]
			})

			// Re-link the winners
			winners = nil
			for i := 0; i < t.maxPointsPerTile; i++ {
				winnerSlice[i].Next = winners
				winners = winnerSlice[i]
			}
			numWinners = t.maxPointsPerTile

			// Create a new losers list from the remaining points
			var newLosers *geom.LinkedPoint
			for i := t.maxPointsPerTile; i < len(winnerSlice); i++ {
				winnerSlice[i].Next = newLosers
				newLosers = winnerSlice[i]
			}
			// Prepend the new losers to the existing losers list
			if losers != nil {
				end := newLosers
				for end.Next != nil {
					end = end.Next
				}
				end.Next = losers
			}
			losers = newLosers
		}

		t.gridSize = currentGridSize
	}

	t.pts = winners
	t.numPoints = numWinners

	// Distribute losers to children
	cur := losers
	for cur != nil {
		next := cur.Next
		cur.Next = nil
		idx := t.getChildrenIndex(cur.Pt)
		if t.childrenPts[idx] == nil {
			t.childrenPts[idx] = cur
		} else {
			cur.Next = t.childrenPts[idx]
			t.childrenPts[idx] = cur
		}
		cur = next
	}

	// Roll up children with too few points, but respect maxPointsPerTile
	for i, c := range t.childrenPts {
		if c == nil {
			continue
		}
		count := 0
		cur := c
		for cur != nil {
			count++
			cur = cur.Next
		}
		if count < t.minPointsPerChildren {
			// Check if there is enough capacity in the parent tile
			capacity := t.maxPointsPerTile - t.numPoints
			if t.maxPointsPerTile == 0 || count <= capacity {
				// Add child points to current node
				end := c
				for end.Next != nil {
					end = end.Next
				}
				end.Next = t.pts
				t.pts = c
				t.numPoints += count
				t.childrenPts[i] = nil // Child is now empty
			}
		}
	}

	t.built = true
	return nil
}

func (t *Node) IsRoot() bool {
	return t.depth == 0
}

func (t *Node) BoundingBox() geom.BoundingBox {
	return t.bounds
}

func (t *Node) ToParentCRS() *model.Transform {
	return t.localToGlobal
}

func (t *Node) Children() [8]tree.Node {
	t.Lock()
	defer t.Unlock()
	if t.childrenBuilt {
		return t.children
	}
	t.children = [8]tree.Node{}
	if !t.built {
		// not built? return nothing
		return t.children
	}
	for i, c := range t.childrenPts {
		if c == nil {
			continue
		}
		v := &Node{
			pts:                  c,
			childrenPts:          [8]*geom.LinkedPoint{},
			bounds:               geom.NewBoundingBoxFromParent(t.bounds, i),
			depth:                t.depth + 1,
			maxDepth:             t.maxDepth,
			gridSize:             t.gridSize / 2,
			childrenBuilt:        false,
			minPointsPerChildren: t.minPointsPerChildren,
			maxPointsPerTile:     t.maxPointsPerTile,
			localToGlobal:        nil,
		}
		// Children MUST be built before returned
		v.Build()
		t.children[i] = tree.Node(v)
	}
	t.childrenBuilt = true
	return t.children
}

func (t *Node) Points() geom.PointList {
	return geom.NewLinkedPointStream(t.pts, t.numPoints)
}

func (t *Node) TotalNumberOfPoints() int {
	return t.totalNumPoints
}

func (t *Node) NumberOfPoints() int {
	return t.numPoints
}

func (t *Node) IsLeaf() bool {
	for _, v := range t.Children() {
		if v != nil {
			return false
		}
	}
	return true
}

func (t *Node) GeometricError() float64 {
	return math.Sqrt(t.gridSize * t.gridSize * 3)
}

func (t *Node) getChildrenIndex(p model.Point) int {
	if float64(p.X) < t.bounds.Xmid && float64(p.Y) < t.bounds.Ymid && float64(p.Z) < t.bounds.Zmid {
		return 0
	} else if float64(p.X) >= t.bounds.Xmid && float64(p.Y) < t.bounds.Ymid && float64(p.Z) < t.bounds.Zmid {
		return 1
	} else if float64(p.X) < t.bounds.Xmid && float64(p.Y) >= t.bounds.Ymid && float64(p.Z) < t.bounds.Zmid {
		return 2
	} else if float64(p.X) >= t.bounds.Xmid && float64(p.Y) >= t.bounds.Ymid && float64(p.Z) < t.bounds.Zmid {
		return 3
	} else if float64(p.X) < t.bounds.Xmid && float64(p.Y) < t.bounds.Ymid && float64(p.Z) >= t.bounds.Zmid {
		return 4
	} else if float64(p.X) >= t.bounds.Xmid && float64(p.Y) < t.bounds.Ymid && float64(p.Z) >= t.bounds.Zmid {
		return 5
	} else if float64(p.X) < t.bounds.Xmid && float64(p.Y) >= t.bounds.Ymid && float64(p.Z) >= t.bounds.Zmid {
		return 6
	}
	return 7
}

func (t *Node) loadPoints(reader las.LasReader, convFactory coor.ConverterFactory, mut mutator.Mutator, ctx context.Context) error {
	l := loader{
		createCoorConverter: convFactory,
		mutator:             mut,
		workers:             t.loadWorkersNumber,
	}
	return l.load(t, reader, ctx)
}

// sampleGrid performs grid sampling on a set of points.
// It returns the points that are kept (winners), the points that are discarded (losers), the number of winners, and the total number of points processed.
func (t *Node) sampleGrid(points *geom.LinkedPoint, gridSize float64) (winners, losers *geom.LinkedPoint, numWinners, totalPointsProcessed int) {
	if points == nil {
		return nil, nil, 0, 0
	}

	// nX, nY, nZ represent the number of grid cells in each direction, should always be >= 1
	nX := math.Ceil((t.bounds.Xmax - t.bounds.Xmin) / gridSize)
	nY := math.Ceil((t.bounds.Ymax - t.bounds.Ymin) / gridSize)
	nZ := math.Ceil((t.bounds.Zmax - t.bounds.Zmin) / gridSize)

	// these are the actual gridSizes after the rounding
	gridSizeX := (t.bounds.Xmax - t.bounds.Xmin) / nX
	gridSizeY := (t.bounds.Ymax - t.bounds.Ymin) / nY
	gridSizeZ := (t.bounds.Zmax - t.bounds.Zmin) / nZ

	type cell struct {
		pt   *geom.LinkedPoint
		dist float64
	}
	grid := map[[3]int32]cell{}

	cur := points
	for cur != nil {
		totalPointsProcessed++
		next := cur.Next
		cur.Next = nil

		// compute 3D integer coordinates of the cell the point falls into
		iX := int32(math.Min(math.Max(1, math.Ceil((float64(cur.Pt.X)-t.bounds.Xmin)/gridSizeX)), float64(nX)))
		iY := int32(math.Min(math.Max(1, math.Ceil((float64(cur.Pt.Y)-t.bounds.Ymin)/gridSizeY)), float64(nY)))
		iZ := int32(math.Min(math.Max(1, math.Ceil((float64(cur.Pt.Z)-t.bounds.Zmin)/gridSizeZ)), float64(nZ)))
		cellIndex := [3]int32{iX, iY, iZ}

		// compute the cell center coordinates
		cX := t.bounds.Xmin + float64(iX-1)*gridSizeX + gridSizeX/2
		cY := t.bounds.Ymin + float64(iY-1)*gridSizeY + gridSizeY/2
		cZ := t.bounds.Zmin + float64(iZ-1)*gridSizeZ + gridSizeZ/2

		curDist := (cX-float64(cur.Pt.X))*(cX-float64(cur.Pt.X)) + (cY-float64(cur.Pt.Y))*(cY-float64(cur.Pt.Y)) + (cZ-float64(cur.Pt.Z))*(cZ-float64(cur.Pt.Z))

		oldWinner, ok := grid[cellIndex]
		if !ok {
			grid[cellIndex] = cell{pt: cur, dist: curDist}
		} else {
			if curDist < oldWinner.dist {
				grid[cellIndex] = cell{pt: cur, dist: curDist}
				oldWinner.pt.Next = losers
				losers = oldWinner.pt
			} else {
				cur.Next = losers
				losers = cur
			}
		}
		cur = next
	}

	for _, c := range grid {
		c.pt.Next = winners
		winners = c.pt
		numWinners++
	}

	return winners, losers, numWinners, totalPointsProcessed
}
