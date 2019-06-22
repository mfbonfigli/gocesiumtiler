package octree

// Contains the options needed for the tiling algorithm
type TilerOptions struct {
	Input                  string         // Input LAS file/folder
	Output                 string         // Output Cesium Tileset folder
	Srid                   int            // EPSG code for SRID of input LAS points
	ZOffset                float64        // Z Offset in meters to apply to points during conversion
	MaxNumPointsPerNode    int32          // Maximum allowed number of points per node
	EnableGeoidZCorrection bool           // Enables the conversion from geoid to ellipsoid height
	FolderProcessing       bool           // Enables the processing of all LAS files in folder
	Recursive              bool           // Recursive lookup of LAS files in subfolders
	Silent                 bool           // Suppressess console messages
	Strategy               LoaderStrategy // Point loading strategy
}
