package main

import (
	"errors"
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/converters"
	"github.com/mfbonfigli/gocesiumtiler/io"
	lidario "github.com/mfbonfigli/gocesiumtiler/lasread"
	"github.com/mfbonfigli/gocesiumtiler/structs/octree"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Starts the tiling process
func RunTiler(opts *octree.TilerOptions) error {

	LogOutput("Preparing list of files to process...")

	// Prepare list of files to process
	lasFiles := make([]string, 0)

	// If folder processing is not enabled then las file is given by -input flag, otherwise look for las in -input folder
	// eventually excluding nested folders if Recursive flag is disabled
	if !opts.FolderProcessing {
		lasFiles = append(lasFiles, opts.Input)
	} else {
		baseInfo, _ := os.Stat(opts.Input)
		err := filepath.Walk(opts.Input, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && !opts.Recursive && !os.SameFile(info, baseInfo) {
				return filepath.SkipDir
			} else {
				if strings.ToLower(filepath.Ext(info.Name())) == ".las" {
					lasFiles = append(lasFiles, path)
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	//fileName := "C:\\Users\\bonfi\\Desktop\\las\\output\\2019-JAN-07_Montesilvano3 Track_B_2019-02-25_11h30_01_331.las"
	// fileName := opts.Input

	// Define elevation (Z) correction algorithm to apply
	zCorrectionAlg := func(lat, lon, z float64) float64 {
		return z + opts.ZOffset
	}
	if opts.EnableGeoidZCorrection {
		// TODO: Configurable cell size
		eFixer := converters.NewElevationFixer(4326, 360/6371000*math.Pi*2)
		zCorrectionAlg = func(lat, lon, z float64) float64 {
			zfix, err := eFixer.GetCorrectedElevation(lat, lon, z)
			if err != nil {
				log.Fatal(err)
			}
			return zfix + opts.ZOffset
		}
	}

	// Define loader strategy
	var loader octree.Loader
	loader = octree.NewRandomLoader()
	if opts.Strategy == octree.BoxedRandom {
		loader = octree.NewRandomBoxLoader()
	}

	// load las points in octree buffer
	for i, fileName := range lasFiles {
		// Create empty octree
		OctTree := octree.NewOctTree(opts)

		// Reading files
		LogOutput("Processing file " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(lasFiles)))
		LogOutput("> reading data from las file...", filepath.Base(fileName))

		err := loadLasInOctree(fileName, OctTree, zCorrectionAlg, opts, loader)
		if err != nil {
			log.Fatal(err)
		}

		// Build tree hierarchical structure
		LogOutput("> building data structure...")
		//err = OctTree.BuildTree()
		err = OctTree.Build(loader)
		if err != nil {
			return err
		}

		LogOutput("> exporting data...")
		nameWext := filepath.Base(fileName)
		extension := filepath.Ext(nameWext)
		name := nameWext[0 : len(nameWext)-len(extension)]
		err = exportOctreeAsTileset(opts, OctTree, name)
		if err != nil {
			return err
		}
		LogOutput("> done processing", filepath.Base(fileName))
		converters.DeallocateProjections()
	}
	return nil
}

// Extracts all the points from the given LAS file and loads them in the given octree
func loadLasInOctree(fileName string, OctTree *octree.OctTree, zCorrectionAlg func(lat, lon, z float64) float64, opts *octree.TilerOptions, loader octree.Loader) error {
	// Read las file and obtaining list of OctElements
	err := readLas(fileName, zCorrectionAlg, opts, loader)
	if err != nil {
		return err
	}
	return nil
}

// Reads the given las file and preloads data in a list of OctElements
func readLas(file string, zCorrection func(lat, lon, z float64) float64, opts *octree.TilerOptions, loader octree.Loader) error {
	var lf *lidario.LasFile
	var err error
	lf, err = lidario.NewLasFileForTiler(file, zCorrection, opts.Srid, loader)
	if err != nil {
		return err
	}
	opts.Srid = 4326
	defer func() { _ = lf.Close() }()
	return nil
}

// Exports the point cloud represented by the given built octree into 3D tiles data structure according to the options
// specified in the TilerOptions instance
func exportOctreeAsTileset(opts *octree.TilerOptions, octree *octree.OctTree, subfolder string) error {
	// if octree is not built, exit
	if !octree.Built {
		return errors.New("octree not built, data structure not initialized")
	}

	// a consumer goroutine per CPU
	numConsumers := runtime.NumCPU()

	// init channel where to submit work with a buffer 5 times greater than the number of consumer
	workchan := make(chan *io.WorkUnit, numConsumers*5)

	// init channel where consumers can eventually submit errors that prevented them to finish the job
	errchan := make(chan error)

	var wg sync.WaitGroup

	// init producer
	wg.Add(1)

	go io.Produce(opts.Output, &octree.RootNode, opts, workchan, &wg, subfolder)

	// init consumers
	for i := 0; i < numConsumers; i++ {
		wg.Add(1)
		go io.Consume(workchan, errchan, &wg)
	}

	// wait for producers and consumers to finish
	wg.Wait()

	// close error chan
	close(errchan)

	// find if there are errors in the error channel buffer
	withErrors := false
	for err := range errchan {
		fmt.Println(err)
		withErrors = true
	}
	if withErrors {
		return errors.New("errors raised during execution. Check console output for details")
	}

	return nil
}
