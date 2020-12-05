package point_loader

import (
	"github.com/mfbonfigli/gocesiumtiler/structs/data"
	"math"
	"math/rand"
	"sync"
)

// Stores Points and returns them shuffled according to the following strategy. Points are grouped in buckets of
// 1e-6 deg of latitude and longitude. Boxes are randomly sorted and the next data is selected at random from the first
// box. Next data is taken at random from the following box. When boxes have all been visited the selection will begin
// again from the first one. If one box becomes empty is removed and replaced with the last one in the set.
type RandomBoxLoader struct {
	sync.Mutex
	Buckets                            map[geoKey]*safeElementList
	Keys                               []*geoKey
	currentKeyIndex                    int64
	minX, maxX, minY, maxY, minZ, maxZ float64
}

// Instances a new RandomBoxLoader
func NewRandomBoxLoader() *RandomBoxLoader {
	return &RandomBoxLoader{
		Buckets:         make(map[geoKey]*safeElementList),
		Keys:            make([]*geoKey, 0),
		currentKeyIndex: 0,
		minX:            math.MaxFloat64,
		minY:            math.MaxFloat64,
		minZ:            math.MaxFloat64,
		maxX:            -1 * math.MaxFloat64,
		maxY:            -1 * math.MaxFloat64,
		maxZ:            -1 * math.MaxFloat64,
	}
}

// Unique spatial key structure for grouping points
type geoKey struct {
	X int
	Y int
	Z int
}

// Mutexed list of pointers to Points for concurrent usage
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

func (eb *RandomBoxLoader) AddElement(e *data.Point) {
	geoKey := computeGeoKey(e)
	eb.Lock()
	eb.recomputeBoundsFromElement(e)
	if bucket := eb.Buckets[geoKey]; bucket == nil {
		eb.Buckets[geoKey] = newSafeElementList()
		eb.Unlock()
	} else {
		eb.Unlock()
		bucket.Lock()
		bucket.Elements = append(bucket.Elements, e)
		bucket.Unlock()
	}
}

func (eb *RandomBoxLoader) GetNext() (*data.Point, bool) {
	eb.Lock()
	defer eb.Unlock()
	if len(eb.Keys) == 0 {
		return nil, false
	}
	key := eb.Keys[eb.currentKeyIndex]
	el, filled := eb.Buckets[*key].removeAndGetFirst()
	if !filled {
		delete(eb.Buckets, *key)
		eb.Keys[eb.currentKeyIndex] = eb.Keys[len(eb.Keys)-1]
		eb.Keys = eb.Keys[:len(eb.Keys)-1]
	}
	eb.currentKeyIndex++
	count := len(eb.Keys)
	if eb.currentKeyIndex > int64(count-1) {
		eb.currentKeyIndex = 0
	}
	return el, count > 0
}

// Initializes the structure to allow proper retrieval of Points. Shuffles the box order and points in each of the boxes.
func (eb *RandomBoxLoader) Initialize() {
	for i, b := range eb.Buckets {
		var j = i
		eb.Keys = append(eb.Keys, &j)
		rand.Shuffle(len(b.Elements), func(i, j int) { b.Elements[i], b.Elements[j] = b.Elements[j], b.Elements[i] })
	}
	rand.Shuffle(len(eb.Keys), func(i, j int) { eb.Keys[i], eb.Keys[j] = eb.Keys[j], eb.Keys[i] })
	eb.currentKeyIndex = 0
}

// Computes the geokey associated to the given Point
func computeGeoKey(e *data.Point) geoKey {
	// 6th decimal for lat lng, 1st decimal for meters
	return geoKey{
		X: int(math.Floor(e.X / 10e-6)),
		Y: int(math.Floor(e.Y / 1 * 10e-6)),
		Z: int(math.Floor(e.Z / 10e-1)),
	}
}

func (eb *RandomBoxLoader) GetBounds() []float64 {
	return []float64{eb.minX, eb.maxX, eb.minY, eb.maxY, eb.minZ, eb.maxZ}
}

// Updates the data cloud bounds according  to the given additional element to insert
func (eb *RandomBoxLoader) recomputeBoundsFromElement(element *data.Point) {
	eb.minX = math.Min(float64(element.X), eb.minX)
	eb.minY = math.Min(float64(element.Y), eb.minY)
	eb.minZ = math.Min(float64(element.Z), eb.minZ)
	eb.maxX = math.Max(float64(element.X), eb.maxX)
	eb.maxY = math.Max(float64(element.Y), eb.maxY)
	eb.maxZ = math.Max(float64(element.Z), eb.maxZ)
}
