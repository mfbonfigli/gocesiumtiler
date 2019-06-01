// Copyright 2019 Massimo Federico Bonfigli

package main

import (
	"cesium_tiler/structs"
	"flag"
	"fmt"
	"os"
)

func main() {
	//defer profile.Start(profile.MemProfile).Stop()

	// Retrieve command line args
	input := flag.String("Input", "", "input las folder")
	output := flag.String("Output", "", "output las folder")
	srid := flag.Int("Srid", 4326, "EPSG srid or input points")
	colorBits := flag.Int("ColorDepth", 8, "Number of bits of color depth")
	zOffset := flag.Float64("Zoffset", 0, "Vertical offset to apply in meters")
	maxNumPts := flag.Int("MaxNumPts", 50000, "Max number of points per tile")
	zGeoidCorrection := flag.Bool("correctGeoidHeight", false, "Enable Geoid to Ellipsoid elevation correction")

	flag.Parse()

	// Put args inside a TilerOptions struct
	opts := structs.TilerOptions{
		Input:                  *input,
		Output:                 *output,
		Srid:                   *srid,
		ColorDepth:             *colorBits,
		ZOffset:                *zOffset,
		MaxNumPointsPerNode:    int32(*maxNumPts),
		EnableGeoidZCorrection: *zGeoidCorrection,
	}

	// Validate TilerOptions
	if msg, res := validateOptions(&opts); !res {
		fmt.Println("Error parsing input parameters: " + msg)
	}

	// Starts the tiler

	err := RunTiler(&opts)
	if err != nil {
		fmt.Println("Error while tiling: ", err)
	}
}

// Validates the input options provided to the command line tool checking
// that input and output folders exists and that the specified color depth
// is valid
func validateOptions(opts *structs.TilerOptions) (string, bool) {
	if _, err := os.Stat(opts.Input); os.IsNotExist(err) {
		return "Input folder not found", false
	}
	if _, err := os.Stat(opts.Output); os.IsNotExist(err) {
		return "Output folder not found", false
	}
	if opts.ColorDepth != 8 && opts.ColorDepth != 16 {
		return "Color depth not supported. Only 8 bits or 16 bits are allowed.", false
	}
	return "", true
}
