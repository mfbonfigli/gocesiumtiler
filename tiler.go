package main

import (
	"errors"
	"fmt"
	"go_cesium_tiler/converters"
	"go_cesium_tiler/io"
	lidario "go_cesium_tiler/lasread"
	"go_cesium_tiler/structs/octree"
	"log"
	"runtime"
	"sync"
)

// Starts the tiling process
func RunTiler(opts *octree.TilerOptions) error {
	//fileName := "C:\\Users\\bonfi\\Desktop\\las\\output\\2019-JAN-07_Montesilvano3 Track_B_2019-02-25_11h30_01_331.las"
	fileName := opts.Input

	// Define elevation (Z) correction algorithm to apply
	zCorrectionAlg := func(lat, lon, z float64) float64 {
		return z + opts.ZOffset
	}
	if opts.EnableGeoidZCorrection {
		// TODO: Using a cell size of 1.0 but should be tuned according to the input coords
		eFixer := converters.NewElevationFixer(opts.Srid, 1.0)
		zCorrectionAlg = func(lat, lon, z float64) float64 {
			zfix, err := eFixer.GetCorrectedElevation(lat, lon, z)
			if err != nil {
				log.Fatal(err)
			}
			return zfix + opts.ZOffset
		}
	}

	// Read las file and obtaining list of OctElements
	pts, err := readLas(fileName, zCorrectionAlg)
	if err != nil {
		return err
	}

	// Create empty octree
	OctTree := octree.NewOctTree(opts)

	// Load items into octree
	err = OctTree.AddItems(pts)
	if err != nil {
		return err
	}

	// Build tree hierarchical structure
	err = OctTree.BuildTree()
	if err != nil {
		return err
	}

	return exportOctreeAsTileset(opts, OctTree)
}

// Reads the given las file and preloads data in a list of OctElements
func readLas(file string, zCorrection func(lat, lon, z float64) float64) ([]octree.OctElement, error) {
	var lf *lidario.LasFile
	var err error
	lf, err = lidario.NewLasFileForTiler(file, zCorrection)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lf.Close() }()
	return lf.GetOctElements(), nil
}

// Exports the point cloud represented by the given built octree into 3D tiles data structure according to the options
// specified in the TilerOptions instance
func exportOctreeAsTileset(opts *octree.TilerOptions, octree *octree.OctTree) error {
	// if octree is not built, exit
	if !octree.Built {
		return errors.New("octree not built, data structure not initialized")
	}

	// a consumer goroutine per CPU
	numConsumers := runtime.NumCPU()

	// init channel where to submit work with a buffer 5 times greater than the number of consumer
	workchan := make(chan *io.WorkUnit, numConsumers*5)

	// init channel where consumers can eventually submit errors that prevented them to finish the job
	errchan := make(chan error, )

	var wg sync.WaitGroup

	// init producer
	wg.Add(1)
	go io.Produce(opts.Output, &octree.RootNode, opts, workchan, &wg)

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

