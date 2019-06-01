package lidario

import (
	"errors"
	"math"
)

// FixedRadiusSearchResult is used to store return values from searches
type FixedRadiusSearchResult struct {
	Index       int
	SquaredDist float64
}

type fixedRadiusSearch struct {
	radius, radiusSquared float64
	lf                    *LasFile
	hm                    map[int64]*frsEntryList
	length                int
	xMin, yMin, zMin      float64
	xMax, yMax, zMax      float64
	nRows, nCols, nLayers int64
	nCellsPerLayer        int64
	threeDMode            bool
}

func build(lf *LasFile, radius float64, threeDMode bool) *fixedRadiusSearch {
	hm := make(map[int64]*frsEntryList)
	frs := fixedRadiusSearch{radius: radius, radiusSquared: radius * radius, lf: lf, hm: hm, length: 0, threeDMode: threeDMode}

	frs.xMin = lf.Header.MinX - radius
	frs.yMin = lf.Header.MinY - radius
	frs.zMin = lf.Header.MinZ - radius
	frs.xMax = lf.Header.MaxX + radius
	frs.yMax = lf.Header.MaxY + radius
	frs.zMax = lf.Header.MaxZ + radius
	frs.nRows = int64((frs.xMax - frs.xMin) / radius)
	frs.nCols = int64((frs.yMax - frs.yMin) / radius)
	frs.nLayers = int64((frs.zMax - frs.zMin) / radius)
	frs.nCellsPerLayer = frs.nRows * frs.nCols

	var k int64
	if !frs.threeDMode {
		// var c, r int
		for i, p := range lf.pointData {
			//c, r = frs.getBinCoordinates(p.X, p.Y)
			k = frs.getCellNum2D(p.X, p.Y)
			if list, ok := frs.hm[k]; ok {
				list.push(i)
			} else {
				list = new(frsEntryList)
				list.push(i)
				frs.hm[k] = list
			}
			frs.length++
		}
	} else {
		for i, p := range lf.pointData {
			k = frs.getCellNum3D(p.X, p.Y, p.Z)
			if list, ok := frs.hm[k]; ok {
				list.push(i)
			} else {
				list = new(frsEntryList)
				list.push(i)
				frs.hm[k] = list
			}
			frs.length++
		}
	}

	return &frs
}

func (frs *fixedRadiusSearch) getCellNum2D(x, y float64) int64 {
	if x < frs.xMin || x > frs.xMax || y < frs.yMin || y > frs.yMax {
		return -1
	}
	return int64(math.Floor((y-frs.yMin)/frs.radius))*frs.nCols + int64(math.Floor((x-frs.xMin)/frs.radius))
}

func (frs *fixedRadiusSearch) getBinCoordinates2D(x, y float64) (column, row int64) {
	column = int64(math.Floor((x - frs.xMin) / frs.radius))
	row = int64(math.Floor((y - frs.yMin) / frs.radius))
	return
}

func (frs *fixedRadiusSearch) getCellNum3D(x, y, z float64) int64 {
	if x < frs.xMin || x > frs.xMax || y < frs.yMin || y > frs.yMax || z < frs.zMin || z > frs.zMax {
		return -1
	}
	col := int64(math.Floor((x - frs.xMin) / frs.radius))
	row := int64(math.Floor((y - frs.yMin) / frs.radius))
	layer := int64(math.Floor((z - frs.zMin) / frs.radius))
	return int64(layer*frs.nCellsPerLayer + row*frs.nCols + col)
}

func (frs *fixedRadiusSearch) getBinCoordinates3D(x, y, z float64) (column, row, layer int64) {
	column = int64(math.Floor((x - frs.xMin) / frs.radius))
	row = int64(math.Floor((y - frs.yMin) / frs.radius))
	layer = int64(math.Floor((z - frs.zMin) / frs.radius))
	return
}

func (frs *fixedRadiusSearch) search2D(x, y float64) *FRSResultList {
	ret := new(FRSResultList)
	if x < frs.xMin || x > frs.xMax || y < frs.yMin || y > frs.yMax {
		return ret
	}
	var ok bool
	var l *frsEntryList
	var entry *frsEntryNode
	var squaredDist float64
	var p PointRecord0
	stCol, stRow := frs.getBinCoordinates2D(x-frs.radius, y-frs.radius)
	endCol, endRow := frs.getBinCoordinates2D(x+frs.radius, y+frs.radius)
	var k int64
	for m := stCol; m <= endCol; m++ {
		for n := stRow; n <= endRow; n++ {
			k = n*frs.nCols + m
			if l, ok = frs.hm[k]; ok {
				for entry = l.first(); entry != nil; entry = entry.next() {
					// calculate the squared distance to (x,y)
					p = frs.lf.pointData[entry.index]
					squaredDist = (x-p.X)*(x-p.X) + (y-p.Y)*(y-p.Y)
					if squaredDist <= frs.radiusSquared {
						ret.Push(FixedRadiusSearchResult{Index: entry.index, SquaredDist: squaredDist})
					}
				}
			}
		}
	}
	return ret

	// for m := -1; m <= 1; m++ {
	// 	for n := -1; n <= 1; n++ {
	// 		if valContainer, ok = frs.hm[frsKey{col: c + m, row: r + n}]; ok {
	// 			for _, entry = range valContainer.data {
	// 				// calculate the squared distance to (x,y)
	// 				squaredDist = (x-entry.x)*(x-entry.x) + (y-entry.y)*(y-entry.y)
	// 				if squaredDist <= frs.radiusSquared {
	// 					ret.Push(FixedRadiusSearchResult{entry: entry, squaredDist: squaredDist})
	// 					// ret = append(ret, FixedRadiusSearchResult{entry: entry, squaredDist: squaredDist})
	// 				}
	// 			}
	// 		}
	// 	}
	// }
}

func (frs *fixedRadiusSearch) search3D(x, y, z float64) *FRSResultList {
	ret := new(FRSResultList)
	if x < frs.xMin || x > frs.xMax || y < frs.yMin || y > frs.yMax || z < frs.zMin || z > frs.zMax {
		return ret
	}
	var ok bool
	var l *frsEntryList
	var entry *frsEntryNode
	var squaredDist float64
	var p PointRecord0

	stCol, stRow, stLayer := frs.getBinCoordinates3D(x-frs.radius, y-frs.radius, z-frs.radius)
	endCol, endRow, endLayer := frs.getBinCoordinates3D(x+frs.radius, y+frs.radius, z+frs.radius)
	var k int64
	for m := stCol; m <= endCol; m++ {
		for n := stRow; n <= endRow; n++ {
			for s := stLayer; s <= endLayer; s++ {
				k = s*frs.nCellsPerLayer + n*frs.nCols + m
				if l, ok = frs.hm[k]; ok {
					for entry = l.first(); entry != nil; entry = entry.next() {
						// calculate the squared distance to (x,y)
						p = frs.lf.pointData[entry.index]
						squaredDist = (x-p.X)*(x-p.X) + (y-p.Y)*(y-p.Y) + (z-p.Z)*(z-p.Z)
						if squaredDist <= frs.radiusSquared {
							ret.Push(FixedRadiusSearchResult{Index: entry.index, SquaredDist: squaredDist})
						}
					}
				}
			}
		}
	}

	return ret
}

// FRSResultNode list node
type FRSResultNode struct {
	FixedRadiusSearchResult // Embedded struct
	next, prev              *FRSResultNode
}

// FRSResultList list return from a fixed-radius search
type FRSResultList struct {
	head, tail *FRSResultNode
	size       int
}

// First returns the head of the list
func (l *FRSResultList) First() *FRSResultNode {
	return l.head
}

// Next returns the next node to the current
func (n *FRSResultNode) Next() *FRSResultNode {
	return n.next
}

// Prev returns the previous node to the current
func (n *FRSResultNode) Prev() *FRSResultNode {
	return n.prev
}

// Len return the list's length
func (l *FRSResultList) Len() int {
	return l.size
}

// Push Create new node with value
func (l *FRSResultList) Push(value FixedRadiusSearchResult) *FRSResultList {
	n := &FRSResultNode{FixedRadiusSearchResult: value}
	if l.size > 0 {
		l.tail.next = n // Add after prev last node
		n.prev = l.tail // Link back to prev last node
	} else {
		l.head = n // First node
	}
	l.tail = n // reset tail to newly added node
	l.size++
	return l
}

var errEmpty = errors.New("ERROR - List is empty")

// Pop last item from list
func (l *FRSResultList) Pop() (value FixedRadiusSearchResult, err error) {
	if l.size > 0 {
		value, l.tail = l.tail.FixedRadiusSearchResult, l.tail.prev
		if l.tail == nil {
			l.head = nil
		}
		l.size--
		return
	}
	return value, errEmpty
}

type frsEntryNode struct {
	index              int //frsEntry           // Embedded struct
	nextNode, prevNode *frsEntryNode
}

// FRSResultList list return from a fixed-radius search
type frsEntryList struct {
	head, tail *frsEntryNode
	size       int
}

// First returns the head of the list
func (l *frsEntryList) first() *frsEntryNode {
	return l.head
}

// Next returns the next node to the current
func (n *frsEntryNode) next() *frsEntryNode {
	return n.nextNode
}

// Prev returns the previous node to the current
func (n *frsEntryNode) prev() *frsEntryNode {
	return n.prevNode
}

// Len return the list's length
func (l *frsEntryList) len() int {
	return l.size
}

// Push Create new node with value
func (l *frsEntryList) push(value int) *frsEntryList {
	n := &frsEntryNode{index: value}
	if l.size > 0 {
		l.tail.nextNode = n // Add after prev last node
		n.prevNode = l.tail // Link back to prev last node
	} else {
		l.head = n // First node
	}
	l.tail = n // reset tail to newly added node
	l.size++
	return l
}
