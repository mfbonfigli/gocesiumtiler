/*
 * This file is part of the Go Cesium Point Cloud Tiler distribution (https://github.com/mfbonfigli/gocesiumtiler).
 * Copyright (c) 2019 Massimo Federico Bonfigli - m.federico.bonfigli@gmail.com
 *
 * This program is free software; you can redistribute it and/or modify it
 * under the terms of the GNU Lesser General Public License Version 3 as
 * published by the Free Software Foundation;
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 *
 * This software also uses third party components. You can find information
 * on their credits and licensing in the file LICENSE-3RD-PARTIES.md that
 * you should have received togheter with the source code.
 */

package main

import (
	"flag"
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/app"
	"github.com/mfbonfigli/gocesiumtiler/converters/gh_ellipsoid_to_geoid_z_converter"
	"github.com/mfbonfigli/gocesiumtiler/converters/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/structs/tiler"
	"github.com/mfbonfigli/gocesiumtiler/utils"
	"log"
	"os"
	"time"
)

const logo = `
                           _                 _   _ _
  __ _  ___   ___ ___  ___(_)_   _ _ __ ___ | |_(_) | ___ _ __ 
 / _  |/ _ \ / __/ _ \/ __| | | | | '_   _ \| __| | |/ _ \ '__|
| (_| | (_) | (_|  __/\__ \ | |_| | | | | | | |_| | |  __/ |   
 \__, |\___/ \___\___||___/_|\__,_|_| |_| |_|\__|_|_|\___|_|   
  __| | A Cesium Point Cloud tile generator written in golang
 |___/  Copyright 2019 - Massimo Federico Bonfigli    
`


func main() {
	//defer profile.Start(profile.CPUProfile).Stop()

	// Retrieve command line args
	flags := utils.ParseFlags()

	// Prints the command line flag description
	if *flags.Help {
		showHelp()
		return
	}

	// set logging and timestamp logging
	if *flags.Silent {
		utils.DisableLogger()
	} else {
		printLogo()
	}
	if !*flags.LogTimestamp {
		utils.DisableLoggerTimestamp()
	}

	// eventually set HQ strategy
	strategy := tiler.FullyRandom
	if *flags.Hq {
		strategy = tiler.BoxedRandom
	}

	// default converter services
	var coordinateConverterService = proj4_coordinate_converter.NewProj4CoordinateConverter()
	var elevationConverterService = gh_ellipsoid_to_geoid_z_converter.NewGHElevationConverter(coordinateConverterService)

	// Put args inside a TilerOptions struct
	opts := tiler.TilerOptions{
		Input:                  *flags.Input,
		Output:                 *flags.Output,
		Srid:                   *flags.Srid,
		ZOffset:                *flags.ZOffset,
		MaxNumPointsPerNode:    int32(*flags.MaxNumPts),
		EnableGeoidZCorrection: *flags.ZGeoidCorrection,
		FolderProcessing:       *flags.FolderProcessing,
		Recursive:              *flags.RecursiveFolderProcessing,
		Silent:                 *flags.Silent,
		Strategy:               strategy,
		CoordinateConverter:    coordinateConverterService,
		ElevationConverter:     elevationConverterService,
	}

	// Validate TilerOptions
	if msg, res := validateOptions(&opts); !res {
		log.Fatal("Error parsing input parameters: " + msg)
	}

	// Starts the tiler
	// defer timeTrack(time.Now(), "tiler")
	err := app.RunTiler(&opts)
	if err != nil {
		log.Fatal("Error while tiling: ", err)
	} else {
		utils.LogOutput("Conversion Completed")
	}
}

// Validates the input options provided to the command line tool checking
// that input and output folders/files exist
func validateOptions(opts *tiler.TilerOptions) (string, bool) {
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
	utils.LogOutput(fmt.Sprintf("%s took %s", name, elapsed))
}

func printLogo() {
	fmt.Println(logo)
}

func showHelp() {
	printLogo()
	fmt.Println("***")
	fmt.Println("GoCesiumTiler is a tool that processes LAS files and transforms them in a 3D Tiles data structure consumable by Cesium.js")
	fmt.Println("***")
	fmt.Println("")
	fmt.Println("Command line flags: ")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.PrintDefaults()
}
