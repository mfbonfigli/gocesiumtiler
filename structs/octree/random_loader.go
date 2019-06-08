package octree

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
)

// Stores OctElements and returns them randomly
type RandomLoader struct {
	sync.Mutex
	fullyRandomList                    []*OctElement
	currentKeyIndex                    int64
	minX, maxX, minY, maxY, minZ, maxZ float64
}

// Instances a new RandomLoader
func NewRandomLoader() *RandomLoader {
	return &RandomLoader{
		currentKeyIndex: 0,
		minX:            math.MaxFloat64,
		minY:            math.MaxFloat64,
		minZ:            math.MaxFloat64,
		maxX:            -1 * math.MaxFloat64,
		maxY:            -1 * math.MaxFloat64,
		maxZ:            -1 * math.MaxFloat64,
	}
}

func (eb *RandomLoader) AddElement(e *OctElement) {
	eb.Lock()
	eb.fullyRandomList = append(eb.fullyRandomList, e)
	eb.recomputeBoundsFromElement(e)
	eb.Unlock()
}

func (eb *RandomLoader) GetNext() (*OctElement, bool) {
	length := len(eb.fullyRandomList)
	counter := int(atomic.AddInt64(&eb.currentKeyIndex, 1))
	if counter > length-1 {
		return nil, false
	} else {
		return eb.fullyRandomList[counter], atomic.LoadInt64(&eb.currentKeyIndex) < int64(length-1)
	}
}

func (eb *RandomLoader) Initialize() {
	rand.Shuffle(len(eb.fullyRandomList), func(i, j int) { eb.fullyRandomList[i], eb.fullyRandomList[j] = eb.fullyRandomList[j], eb.fullyRandomList[i] })
	eb.currentKeyIndex = -1
}

// Updates the point cloud bounds as per loaded RandomLoader elements and given additional element
func (eb *RandomLoader) recomputeBoundsFromElement(element *OctElement) {
	eb.minX = math.Min(float64(element.X), eb.minX)
	eb.minY = math.Min(float64(element.Y), eb.minY)
	eb.minZ = math.Min(float64(element.Z), eb.minZ)
	eb.maxX = math.Max(float64(element.X), eb.maxX)
	eb.maxY = math.Max(float64(element.Y), eb.maxY)
	eb.maxZ = math.Max(float64(element.Z), eb.maxZ)
}

func (eb *RandomLoader) GetBounds() []float64 {
	return []float64{eb.minX, eb.maxX, eb.minY, eb.maxY, eb.minZ, eb.maxZ}
}
