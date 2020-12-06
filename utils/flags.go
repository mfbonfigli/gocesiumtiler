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
}

func ParseFlags() Flags {
	input := flag.String("input", "", "Specifies the input las file/folder.")
	output := flag.String("output", "", "Specifies the output folder where to write the tileset data.")
	srid := flag.Int("srid", 4326, "EPSG srid code of input points.")
	zOffset := flag.Float64("zoffset", 0, "Vertical offset to apply to points, in meters.")
	maxNumPts := flag.Int("maxpts", 50000, "Max number of points per tile. ")
	zGeoidCorrection := flag.Bool("geoid", false, "Enables Geoid to Ellipsoid elevation correction. Use this flag if your input LAS files have Z coordinates specified relative to the Earth geoid rather than to the standard ellipsoid.")
	folderProcessing := flag.Bool("folder", false, "Enables processing of all las files from input folder. Input must be a folder if specified")
	recursiveFolderProcessing := flag.Bool("recursive", false, "Enables recursive lookup for all .las files inside the subfolders")
	silent := flag.Bool("silent", false, "Use to suppress all the non-error messages.")
	logTimestamp := flag.Bool("timestamp", false, "Adds timestamp to log messages.")
	hq := flag.Bool("hq", false, "Enables a higher quality random pick algorithm.")
	help := flag.Bool("help", false, "Displays this help.")

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
	}
}
