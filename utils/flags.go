package utils

import "flag"

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
	Hq                        *bool
	Help                      *bool
	Version                   *bool
}

func ParseFlags() Flags {
	input := defineStringFlag("input", "i", "", "Specifies the input las file/folder.")
	output := defineStringFlag("output", "o", "", "Specifies the output folder where to write the tileset data.")
	srid := defineIntFlag("srid", "e", 4326, "EPSG srid code of input points.")
	zOffset := defineFloat64Flag("zoffset", "z", 0, "Vertical offset to apply to points, in meters.")
	maxNumPts := defineIntFlag("maxpts", "m", 50000, "Max number of points per tile. ")
	zGeoidCorrection := defineBoolFlag("geoid", "g", false, "Enables Geoid to Ellipsoid elevation correction. Use this flag if your input LAS files have Z coordinates specified relative to the Earth geoid rather than to the standard ellipsoid.")
	folderProcessing := defineBoolFlag("folder", "f", false, "Enables processing of all las files from input folder. Input must be a folder if specified")
	recursiveFolderProcessing := defineBoolFlag("recursive", "r", false, "Enables recursive lookup for all .las files inside the subfolders")
	silent := defineBoolFlag("silent", "s", false, "Use to suppress all the non-error messages.")
	logTimestamp := defineBoolFlag("timestamp", "t", false, "Adds timestamp to log messages.")
	hq := defineBoolFlag("hq", "hq", false, "Enables a higher quality random pick algorithm.")
	help := defineBoolFlag("help", "h", false, "Displays this help.")
	version := defineBoolFlag("version", "v", false, "Displays the version of gocesiumtiler.")

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
		Hq:                        hq,
		Help:                      help,
		Version:                   version,
	}
}

func defineStringFlag(name string, shortHand string, defaultValue string, usage string) *string {
	var output string
	flag.StringVar(&output, name, defaultValue, usage)
	if shortHand != name {
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
