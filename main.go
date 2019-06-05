// Copyright 2019 Massimo Federico Bonfigli

package main

import (
	"flag"
	"fmt"
	"go_cesium_tiler/structs/octree"
	"log"
	"os"
	"time"
)

var log_enabled = true
var timestamp_enabled = true

func main() {
	//defer profile.Start(profile.CPUProfile).Stop()

	// Retrieve command line args
	input := flag.String("input", "", "input las file/folder")
	output := flag.String("output", "", "output folder")
	srid := flag.Int("srid", 4326, "EPSG srid or input points")
	zOffset := flag.Float64("zoffset", 0, "Vertical offset to apply in meters")
	maxNumPts := flag.Int("maxpts", 50000, "Max number of points per tile")
	zGeoidCorrection := flag.Bool("geoid", false, "Enables Geoid to Ellipsoid elevation correction")
	folderProcessing := flag.Bool("folder", false, "Enables processing of all las files from input folder. Input must be a folder if specified")
	recursiveFolderProcessing := flag.Bool("recursive", false, "Enables recursive lookup for all .las files inside the subfolders")
	silent := flag.Bool("silent", false, "suppresses all the non-error messages")
	logTimestamp := flag.Bool("timestamp", false, "adds timestamp to log messages")
	help := flag.Bool("help", false, "prints the help")

	flag.Parse()

	// Prints the command line flag description
	if *help {
		fmt.Println("* Cesium Point Cloud Tiler *")
		fmt.Println("* Copyright 2019 Massimo Federico Bonfigli *")
		fmt.Println("* ")
		fmt.Println("* a command line tool for generating cesium 3D tiles of point clouds from LAS files")
		fmt.Println("")
		fmt.Println("")
		fmt.Println("Command line flags: ")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		return
	}

	// set logging and timestamp logging
	log_enabled = !*silent
	timestamp_enabled = *logTimestamp

	// Put args inside a TilerOptions struct
	opts := octree.TilerOptions{
		Input:                  *input,
		Output:                 *output,
		Srid:                   *srid,
		ZOffset:                *zOffset,
		MaxNumPointsPerNode:    int32(*maxNumPts),
		EnableGeoidZCorrection: *zGeoidCorrection,
		FolderProcessing:       *folderProcessing,
		Recursive:              *recursiveFolderProcessing,
		Silent:                 *silent,
	}

	// Validate TilerOptions
	if msg, res := validateOptions(&opts); !res {
		log.Fatal("Error parsing input parameters: " + msg)
	}

	// Starts the tiler
	defer timeTrack(time.Now(), "tiler")
	err := RunTiler(&opts)
	if err != nil {
		log.Fatal("Error while tiling: ", err)
	} else {
		LogOutput("Conversion Completed")
	}
}

// Validates the input options provided to the command line tool checking
// that input and output folders exists and that the specified color depth
// is valid
func validateOptions(opts *octree.TilerOptions) (string, bool) {
	if _, err := os.Stat(opts.Input); os.IsNotExist(err) {
		return "Input file/folder not found", false
	}
	if _, err := os.Stat(opts.Output); os.IsNotExist(err) {
		return "Output folder not found", false
	}
	return "", true
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	if log_enabled {
		log.Printf("%s took %s", name, elapsed)
	}
}

func LogOutput(val ...interface{}) {
	if log_enabled {
		if timestamp_enabled {
			fmt.Print("[" + time.Now().Format("2006-01-02 15.04:05.000") + "] ")
			fmt.Println(val...)
		} else {
			fmt.Println(val...)
		}
	}
}
