package pkg

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/io"
	"github.com/mfbonfigli/gocesiumtiler/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"github.com/mfbonfigli/gocesiumtiler/pkg/algorithm_manager"
	"github.com/mfbonfigli/gocesiumtiler/tools"
)

type ITiler interface {
	RunTiler(opts *tiler.TilerOptions) error
}

type Tiler struct {
	fileFinder       tools.FileFinder
	algorithmManager algorithm_manager.AlgorithmManager
}

func NewTiler(fileFinder tools.FileFinder, algorithmManager algorithm_manager.AlgorithmManager) ITiler {
	return &Tiler{
		fileFinder:       fileFinder,
		algorithmManager: algorithmManager,
	}
}

// Starts the tiling process
func (tiler *Tiler) RunTiler(opts *tiler.TilerOptions) error {
	tools.LogOutput("Preparing list of files to process...")

	// Prepare list of files to process
	lasFiles := tiler.fileFinder.GetLasFilesToProcess(opts)

	// load las points in octree buffer
	for i, filePath := range lasFiles {
		// Define point_loader strategy
		var tree = tiler.algorithmManager.GetTreeAlgorithm()
		tools.LogOutput("Processing file " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(lasFiles)))
		tiler.processLasFile(filePath, opts, tree)
	}
	tiler.algorithmManager.GetCoordinateConverterAlgorithm().Cleanup()

	return nil
}

func (tiler *Tiler) processLasFile(filePath string, opts *tiler.TilerOptions, tree octree.ITree) {
	// Create empty octree
	r, err := tiler.getLasReader(filePath, opts)
	if err != nil {
		log.Fatal(err)
	}
	tiler.prepareDataStructure(tree, r)
	tiler.exportToCesiumTileset(tree, opts, getFilenameWithoutExtension(filePath))

	tools.LogOutput("> done processing", filepath.Base(filePath))
}

func (tiler *Tiler) prepareDataStructure(octree octree.ITree, r las.LasReader) {
	// Build tree hierarchical structure
	tools.LogOutput("> building data structure...")
	err := octree.Build(r)

	if err != nil {
		log.Fatal(err)
	}
}

func (tiler *Tiler) exportToCesiumTileset(octree octree.ITree, opts *tiler.TilerOptions, fileName string) {
	tools.LogOutput("> exporting data...")
	err := tiler.exportTreeAsTileset(opts, octree, fileName)
	if err != nil {
		log.Fatal(err)
	}
}

func getFilenameWithoutExtension(filePath string) string {
	nameWext := filepath.Base(filePath)
	extension := filepath.Ext(nameWext)
	return nameWext[0 : len(nameWext)-len(extension)]
}

// Reads the given las file and returns a point reader
func (tiler *Tiler) getLasReader(file string, opts *tiler.TilerOptions) (las.LasReader, error) {
	tools.LogOutput("> parsing las file...", filepath.Base(file))
	return las.NewFileLasReader(file, opts.Srid, opts.EightBitColors)
}

// Exports the data cloud represented by the given built octree into 3D tiles data structure according to the options
// specified in the TilerOptions instance
func (tiler *Tiler) exportTreeAsTileset(opts *tiler.TilerOptions, octree octree.ITree, subfolder string) error {
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

	producer := io.NewStandardProducer(opts.Output, subfolder, opts, octree)
	go producer.Produce(workChannel, &waitGroup, octree.GetRootNode())

	// add consumers to waitgroup and launch them
	for i := 0; i < numConsumers; i++ {
		waitGroup.Add(1)
		consumer := io.NewStandardConsumer(tiler.algorithmManager.GetCoordinateConverterAlgorithm(), opts.RefineMode)
		go consumer.Consume(workChannel, errorChannel, &waitGroup)
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
