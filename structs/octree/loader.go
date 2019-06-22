package octree

type LoaderStrategy int

const (
	// Uniform random pick among all loaded elements. Points will tend to be selected in areas with higher density.
	FullyRandom LoaderStrategy = 0

	// Uniform pick in small boxes of points randomly ordered. Point will tend to be more evenly spaced at lower zoom levels.
	// Points are grouped in buckets of 1e-6 deg of latitude and longitude. Boxes are randomly sorted and the next point
	// is selected at random from the first box. Next point is taken at random from the following box. When boxes have all been visited
	// the selection will begin again from the first one. If one box becomes empty is removed and replaced with the last one in the set.
	BoxedRandom LoaderStrategy = 1
)

// A Loader contains methods to store and properly shuffle OctElements for subsequent retrieval in the generation of the
// tree structure
type Loader interface {
	// Adds an OctElement to the Loader
	AddElement(e *OctElement)

	// Returns the next random OctElement from the Loader
	GetNext() (*OctElement, bool)

	// Initializes the structure to allow proper retrieval of OctElements. Must be called after last element has been added but
	// before first call to GetNext
	Initialize()

	// Returns the bounding box extremes of the stored cloud minX, maxX, minY, maxY, minZ, maxZ
	GetBounds() []float64
}
