package point_loader

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/mfbonfigli/gocesiumtiler/internal/data"
)

// Stores points and returns them in order
type SequentialLoader struct {
	sync.Mutex
	sequentialList                     []*data.Point
	currentKeyIndex                    int64
	minX, maxX, minY, maxY, minZ, maxZ float64
}

// Instances a new SequentialLoader
func NewSequentialLoader() *SequentialLoader {
	return &SequentialLoader{
		currentKeyIndex: -1,
		minX:            math.MaxFloat64,
		minY:            math.MaxFloat64,
		minZ:            math.MaxFloat64,
		maxX:            -1 * math.MaxFloat64,
		maxY:            -1 * math.MaxFloat64,
		maxZ:            -1 * math.MaxFloat64,
	}
}

func (eb *SequentialLoader) AddPoint(e *data.Point) {
	eb.Lock()
	eb.sequentialList = append(eb.sequentialList, e)
	eb.recomputeBoundsFromElement(e)
	eb.Unlock()
}

func (eb *SequentialLoader) GetNext() (*data.Point, bool) {
	length := len(eb.sequentialList)
	counter := int(atomic.AddInt64(&eb.currentKeyIndex, 1))
	if counter > length-1 {
		// marks the slice nil for garbage collection
		eb.sequentialList = nil
		return nil, false
	} else {
		value := eb.sequentialList[counter]
		// deallocates the pointer for GC purposes
		eb.sequentialList[counter] = nil
		return value, atomic.LoadInt64(&eb.currentKeyIndex) < int64(length-1)
	}
}

func (eb *SequentialLoader) InitializeLoader() {}

// Updates the data cloud bounds as per loaded RandomLoader elements and given additional element
func (eb *SequentialLoader) recomputeBoundsFromElement(element *data.Point) {
	eb.minX = math.Min(float64(element.X), eb.minX)
	eb.minY = math.Min(float64(element.Y), eb.minY)
	eb.minZ = math.Min(float64(element.Z), eb.minZ)
	eb.maxX = math.Max(float64(element.X), eb.maxX)
	eb.maxY = math.Max(float64(element.Y), eb.maxY)
	eb.maxZ = math.Max(float64(element.Z), eb.maxZ)
}

func (eb *SequentialLoader) GetBounds() []float64 {
	return []float64{eb.minX, eb.maxX, eb.minY, eb.maxY, eb.minZ, eb.maxZ}
}
