package tools

import (
	"flag"
)

type Flags struct {
	Input                     *string
	Output                    *string
	Srid                      *int
	ZOffset                   *float64
	MaxNumPts                 *int
	ZGeoidCorrection          *bool
	FolderProcessing          *bool
	RecursiveFolderProcessing *bool
	Silent                    *bool
	LogTimestamp              *bool
	Algorithm                 *string
	GridCellMaxSize           *float64
	GridCellMinSize           *float64
	RefineMode                *string
	Help                      *bool
	Version                   *bool
	RootGeometricError		  *float64
}

func ParseFlags() Flags {
	input := defineStringFlag("input", "i", "", "Specifies the input las file/folder.")
	output := defineStringFlag("output", "o", "", "Specifies the output folder where to write the tileset data.")
	srid := defineIntFlag("srid", "e", 4326, "EPSG srid code of input points.")
	zOffset := defineFloat64Flag("zoffset", "z", 0, "Vertical offset to apply to points, in meters.")
	maxNumPts := defineIntFlag("maxpts", "m", 50000, "Max number of points per tile for the Random and RandomBox algorithms.")
	zGeoidCorrection := defineBoolFlag("geoid", "g", false, "Enables Geoid to Ellipsoid elevation correction. Use this flag if your input LAS files have Z coordinates specified relative to the Earth geoid rather than to the standard ellipsoid.")
	folderProcessing := defineBoolFlag("folder", "f", false, "Enables processing of all las files from input folder. Input must be a folder if specified")
	recursiveFolderProcessing := defineBoolFlag("recursive", "r", false, "Enables recursive lookup for all .las files inside the subfolders")
	silent := defineBoolFlag("silent", "s", false, "Use to suppress all the non-error messages.")
	logTimestamp := defineBoolFlag("timestamp", "t", false, "Adds timestamp to log messages.")
	algorithm := defineStringFlag("algorithm", "a", "grid", "Sets the algorithm to use. Must be one of Grid,Random,RandomBox. Grid algorithm is highly suggested, others are deprecated and will be removed in future versions.")
	gridCellMaxSize := defineFloat64Flag("grid-max-size", "x", 5.0, "Max cell size in meters for the grid algorithm. It roughly represents the max spacing between any two samples. ")
	gridCellMinSize := defineFloat64Flag("grid-min-size", "n", 0.15, "Min cell size in meters for the grid algorithm. It roughly represents the minimum possible size of a 3d tile. ")
	refineMode := defineStringFlag("refine-mode", "", "ADD", "Type of refine mode, can be 'ADD' or 'REPLACE'. 'ADD' means that child tiles will not contain the parent tiles points. 'REPLACE' means that they will also contain the parent tiles points. ADD implies less disk space but more network overhead when fetching the data, REPLACE is the opposite.")
	help := defineBoolFlag("help", "h", false, "Displays this help.")
	version := defineBoolFlag("version", "v", false, "Displays the version of gocesiumtiler.")
	rootGeometricError := defineFloat64Flag("root-geometric-error", "k", 1, "Multiplies the geometric error of the root by the given factor. Use this flag if you want to display the tiles in higher zoom levels") 

	flag.Parse()

	return Flags{
		Input:                     input,
		Output:                    output,
		Srid:                      srid,
		ZOffset:                   zOffset,
		MaxNumPts:                 maxNumPts,
		ZGeoidCorrection:          zGeoidCorrection,
		FolderProcessing:          folderProcessing,
		RecursiveFolderProcessing: recursiveFolderProcessing,
		Silent:                    silent,
		LogTimestamp:              logTimestamp,
		Algorithm:                 algorithm,
		GridCellMaxSize:           gridCellMaxSize,
		GridCellMinSize:           gridCellMinSize,
		RefineMode:                refineMode,
		Help:                      help,
		Version:                   version,
		RootGeometricError:		   rootGeometricError,
	}
}

func defineStringFlag(name string, shortHand string, defaultValue string, usage string) *string {
	var output string
	flag.StringVar(&output, name, defaultValue, usage)
	if shortHand != name && shortHand != "" {
		flag.StringVar(&output, shortHand, defaultValue, usage+" (shorthand for "+name+")")
	}

	return &output
}

func defineIntFlag(name string, shortHand string, defaultValue int, usage string) *int {
	var output int
	flag.IntVar(&output, name, defaultValue, usage)
	if shortHand != name {
		flag.IntVar(&output, shortHand, defaultValue, usage+" (shorthand for "+name+")")
	}

	return &output
}

func defineFloat64Flag(name string, shortHand string, defaultValue float64, usage string) *float64 {
	var output float64
	flag.Float64Var(&output, name, defaultValue, usage)
	if shortHand != name {
		flag.Float64Var(&output, shortHand, defaultValue, usage+" (shorthand for "+name+")")
	}
	return &output
}

func defineBoolFlag(name string, shortHand string, defaultValue bool, usage string) *bool {
	var output bool
	flag.BoolVar(&output, name, defaultValue, usage)
	if shortHand != name {
		flag.BoolVar(&output, shortHand, defaultValue, usage+" (shorthand for "+name+")")
	}
	return &output
}
