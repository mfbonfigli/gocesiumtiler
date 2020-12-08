package point_loader

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
)

// A Loader contains methods to store and properly shuffle points for subsequent retrieval in the generation of the
// tree structure
type Loader interface {
	// Adds a Point to the Loader
	AddPoint(e *data.Point)

	// Returns the next random Point from the Loader
	GetNext() (*data.Point, bool)

	// Initializes the structure to allow proper retrieval of points. Must be called after last element has been added but
	// before first call to GetNext
	Initialize()

	// Returns the bounding box extremes of the stored cloud minX, maxX, minY, maxY, minZ, maxZ
	GetBounds() []float64
}
