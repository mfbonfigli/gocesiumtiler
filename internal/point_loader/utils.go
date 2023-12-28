package point_loader

import (
	"math"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/data"
)

// Unique spatial key structure for grouping points
type geoKey struct {
	X int
	Y int
	Z int
}

// Mutexed list of pointers to points for concurrent usage
type safeElementList struct {
	sync.Mutex
	Elements []*data.Point
}

// Instances a new safeElementList
func newSafeElementList() *safeElementList {
	return &safeElementList{
		Elements: make([]*data.Point, 0),
	}
}

// Thread safe removal and restitution of the first element of the safeElementList. Returns also a boolean flag that
// tells the caller if the list is now empty after this retrieval
func (sel *safeElementList) removeAndGetFirst() (*data.Point, bool) {
	var el *data.Point
	var stillItems = false
	sel.Lock()
	num := len(sel.Elements)
	if num > 0 {
		el = sel.Elements[0]
		sel.Elements = sel.Elements[1:]
		if num > 1 {
			stillItems = true
		}
	}
	sel.Unlock()
	return el, stillItems
}

// Computes the geokey associated to the given Point
func computeGeoKey(e *data.Point) geoKey {
	// 6th decimal for lat lng, 1st decimal for meters
	return geoKey{
		X: int(math.Floor(float64(e.X) / 10e-6)),
		Y: int(math.Floor(float64(e.Y) / 1 * 10e-6)),
		Z: int(math.Floor(float64(e.Z) / 10e-1)),
	}
}
