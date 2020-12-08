package tiler

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
)

type Strategy int

const (
	// Uniform random pick among all loaded elements. points will tend to be selected in areas with higher density.
	FullyRandom Strategy = 0

	// Uniform pick in small boxes of points randomly ordered. Point will tend to be more evenly spaced at lower zoom levels.
	// points are grouped in buckets of 1e-6 deg of latitude and longitude. Boxes are randomly sorted and the next data
	// is selected at random from the first box. Next data is taken at random from the following box. When boxes have all been visited
	// the selection will begin again from the first one. If one box becomes empty is removed and replaced with the last one in the set.
	BoxedRandom Strategy = 1
)

// Contains the options needed for the tiling algorithm
type TilerOptions struct {
	Input                  string                                // Input LAS file/folder
	Output                 string                                // Output Cesium Tileset folder
	Srid                   int                                   // EPSG code for SRID of input LAS points
	ZOffset                float64                               // Z Offset in meters to apply to points during conversion
	MaxNumPointsPerNode    int32                                 // Maximum allowed number of points per node
	EnableGeoidZCorrection bool                                  // Enables the conversion from geoid to ellipsoid height
	FolderProcessing       bool                                  // Enables the processing of all LAS files in folder
	Recursive              bool                                  // Recursive lookup of LAS files in subfolders
	Silent                 bool                                  // Suppressess console messages
	Strategy               Strategy                              // Point loading strategy
	CoordinateConverter    converters.CoordinateConverter        // Coordinate converter algorithm
	ElevationConverter     converters.EllipsoidToGeoidZConverter // Elevation converter algorithm
}
