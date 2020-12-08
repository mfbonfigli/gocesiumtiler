package pkg

import (
	"errors"
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_elevation_corrector"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/offset_elevation_corrector"
	"github.com/mfbonfigli/gocesiumtiler/internal/io"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"github.com/mfbonfigli/gocesiumtiler/third_party/lasread"
	"github.com/mfbonfigli/gocesiumtiler/tools"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Starts the tiling process
func RunTiler(opts *tiler.TilerOptions) error {
	tools.LogOutput("Preparing list of files to process...")

	// Prepare list of files to process
	lasFiles := getLasFilesToProcess(opts)

	// Define elevation (Z) correction algorithm to apply
	elevationCorrectionAlg := getElevationCorrectionAlgorithm(opts)

	// Define point_loader strategy
	var tree = getTreeFromOptions(opts)

	// load las points in octree buffer
	for i, filePath := range lasFiles {
		tools.LogOutput("Processing file " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(lasFiles)))
		processLasFile(filePath, opts, tree, elevationCorrectionAlg)
	}

	return nil
}

func processLasFile(filePath string, opts *tiler.TilerOptions, tree octree.ITree, elevationCorrectionAlg converters.ElevationCorrector) {
	// Create empty octree
	readLasData(filePath, elevationCorrectionAlg, opts, tree)
	prepareDataStructure(tree)
	exportToCesiumTileset(tree, opts, getFilenameWithoutExtension(filePath))

	tools.LogOutput("> done processing", filepath.Base(filePath))
	opts.CoordinateConverter.Cleanup()
}

func readLasData(filePath string, elevationCorrectionAlg converters.ElevationCorrector, opts *tiler.TilerOptions, tree octree.ITree) {
	// Reading files
	tools.LogOutput("> reading data from las file...", filepath.Base(filePath))
	err := readLas(filePath, elevationCorrectionAlg, opts, tree)

	if err != nil {
		log.Fatal(err)
	}
}

func prepareDataStructure(octree octree.ITree) {
	// Build tree hierarchical structure
	tools.LogOutput("> building data structure...")
	err := octree.Build()

	if err != nil {
		log.Fatal(err)
	}
}

func exportToCesiumTileset(octree octree.ITree, opts *tiler.TilerOptions, fileName string) {
	tools.LogOutput("> exporting data...")
	err := exportTreeAsTileset(opts, octree, fileName)
	if err != nil {
		log.Fatal(err)
	}
}

func getFilenameWithoutExtension(filePath string) string {
	nameWext := filepath.Base(filePath)
	extension := filepath.Ext(nameWext)
	return nameWext[0 : len(nameWext)-len(extension)]
}

func getTreeFromOptions(options *tiler.TilerOptions) octree.ITree {
	switch options.Strategy {
	case tiler.BoxedRandom:
		return octree.NewBoxedRandomTree(options)
	case tiler.FullyRandom:
		return octree.NewRandomTree(options)
	}

	log.Fatal("Unrecognized strategy")
	return nil
}

func getElevationCorrectionAlgorithm(opts *tiler.TilerOptions) converters.ElevationCorrector {
	if !opts.EnableGeoidZCorrection {
		return offset_elevation_corrector.NewOffsetElevationCorrector(opts.ZOffset)
	} else {
		return geoid_elevation_corrector.NewGeoidElevationCorrector(opts.ZOffset, opts.ElevationConverter)
	}
}

func getLasFilesToProcess(opts *tiler.TilerOptions) []string {
	// If folder processing is not enabled then las file is given by -input flag, otherwise look for las in -input folder
	// eventually excluding nested folders if Recursive flag is disabled
	if !opts.FolderProcessing {
		return []string{opts.Input}
	}

	return getLasFilesFromInputFolder(opts)
}

func getLasFilesFromInputFolder(opts *tiler.TilerOptions) []string {
	var lasFiles = make([]string, 0)

	baseInfo, _ := os.Stat(opts.Input)
	err := filepath.Walk(
		opts.Input,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && !opts.Recursive && !os.SameFile(info, baseInfo) {
				return filepath.SkipDir
			} else {
				if strings.ToLower(filepath.Ext(info.Name())) == ".las" {
					lasFiles = append(lasFiles, path)
				}
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	return lasFiles
}

// Reads the given las file and preloads data in a list of Point
func readLas(file string, zCorrection converters.ElevationCorrector, opts *tiler.TilerOptions, tree octree.ITree) error {
	var lf *lidario.LasFile
	var err error
	var lasFileLoader = lidario.NewLasFileLoader(opts.CoordinateConverter, opts.ElevationConverter, tree)
	lf, err = lasFileLoader.LoadLasFile(file, zCorrection, opts.Srid)
	if err != nil {
		return err
	}
	opts.Srid = 4326
	defer func() { _ = lf.Close() }()
	return nil
}

// Exports the data cloud represented by the given built octree into 3D tiles data structure according to the options
// specified in the TilerOptions instance
func exportTreeAsTileset(opts *tiler.TilerOptions, octree octree.ITree, subfolder string) error {
	// if octree is not built, exit
	if !octree.IsBuilt() {
		return errors.New("octree not built, data structure not initialized")
	}

	// a consumer goroutine per CPU
	numConsumers := runtime.NumCPU()

	// init channel where to submit work with a buffer 5 times greater than the number of consumer
	workChannel := make(chan *io.WorkUnit, numConsumers*5)

	// init channel where consumers can eventually submit errors that prevented them to finish the job
	errorChannel := make(chan error)

	var waitGroup sync.WaitGroup

	// add producer to waitgroup and launch producer goroutine
	waitGroup.Add(1)
	go io.Produce(opts.Output, octree.GetRootNode(), opts, workChannel, &waitGroup, subfolder)

	// add consumers to waitgroup and launch them
	for i := 0; i < numConsumers; i++ {
		waitGroup.Add(1)
		go io.Consume(workChannel, errorChannel, &waitGroup, opts.CoordinateConverter)
	}

	// wait for producers and consumers to finish
	waitGroup.Wait()

	// close error chan
	close(errorChannel)

	// find if there are errors in the error channel buffer
	withErrors := false
	for err := range errorChannel {
		fmt.Println(err)
		withErrors = true
	}
	if withErrors {
		return errors.New("errors raised during execution. Check console output for details")
	}

	return nil
}
