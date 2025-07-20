package tiler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/conv/coor"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/conv/coor/proj"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree/grid"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/writer"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/mutator"
)

type Tiler interface {
	ProcessFiles(inputLasFiles []string, outputFolder string, sourceCRS string, opts *TilerOptions, ctx context.Context) error
	ProcessFolder(inputFolder, outputFolder string, sourceCRS string, opts *TilerOptions, ctx context.Context) error
}

// GoCesiumTiler wraps the logic required to convert
// LAS point clouds into Cesium 3D tiles
type GoCesiumTiler struct {
	convFactory coor.ConverterFactory
	treeProvider
	writerProvider
	lasReaderProvider
}

type treeProvider func(opts *TilerOptions) tree.Tree
type writerProvider func(folder string, opts *TilerOptions) (writer.Writer, error)
type lasReaderProvider func(inputLasFiles []string, sourceCRS string, eightbit bool) (las.LasReader, error)

// NewGoCesiumTiler returns a new tiler to be used to convert LAS files into Cesium 3D Tiles
func NewGoCesiumTiler() (*GoCesiumTiler, error) {
	return &GoCesiumTiler{
		convFactory: func() (coor.Converter, error) {
			return proj.NewProjCoordinateConverter()
		},
		treeProvider: func(opts *TilerOptions) tree.Tree {
			return grid.NewTree(
				grid.WithGridSize(opts.gridSize),
				grid.WithMaxDepth(opts.maxDepth),
				grid.WithLoadWorkersNumber(opts.numWorkers),
				grid.WithMinPointsPerChildren(opts.minPointsPerTile),
				grid.WithMaxPointsPerTile(opts.MaxPointsPerTile),
			)
		},
		writerProvider: func(folder string, opts *TilerOptions) (writer.Writer, error) {
			return writer.NewWriter(folder,
				writer.WithNumWorkers(opts.numWorkers),
				writer.WithTilesetVersion(opts.version),
			)
		},
		lasReaderProvider: func(inputLasFiles []string, sourceCRS string, eightbit bool) (las.LasReader, error) {
			return las.NewCombinedFileLasReader(inputLasFiles, sourceCRS, eightbit)
		},
	}, nil
}

// ProcessFolder converts all LAS files found in the provided input folder converting them into separate tilesets
// each tileset is stored in a subdirectory in the outputFolder named after the filename.
// If sourceCRS is left empty, the CRS will attempted to be autodetected from LAS GeoTIFF or WKT VLRs.
func (t *GoCesiumTiler) ProcessFolder(inputFolder, outputFolder string, sourceCRS string, opts *TilerOptions, ctx context.Context) error {
	files, err := utils.FindLasFilesInFolder(inputFolder)
	if err != nil {
		return err
	}
	for _, f := range files {
		subfolderName := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		err := t.ProcessFiles([]string{f}, filepath.Join(outputFolder, subfolderName), sourceCRS, opts, ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProcessFiles converts the specified LAS files as a single cesium tileset and stores them in the given output folder.
// If sourceCRS is left empty, the CRS will attempted to be autodetected from LAS GeoTIFF or WKT VLRs.
func (t *GoCesiumTiler) ProcessFiles(inputLasFiles []string, outputFolder string, sourceCRS string, opts *TilerOptions, ctx context.Context) error {
	start := time.Now()
	tr := t.treeProvider(opts)
	defer tr.Dispose()

	inputDesc := fmt.Sprintf("%d files", len(inputLasFiles))
	if len(inputLasFiles) == 1 {
		inputDesc = inputLasFiles[0]
	}

	// PARSE LAS HEADER
	emitEvent(EventReadLasHeaderStarted, opts, start, inputDesc, "start reading las")
	lasFile, err := t.lasReaderProvider(inputLasFiles, sourceCRS, opts.eightBitColors)
	if err != nil {
		emitEvent(EventReadLasHeaderError, opts, start, inputDesc, fmt.Sprintf("las read error: %v", err))
		return err
	}
	emitEvent(EventReadLasHeaderCompleted, opts, start, inputDesc, fmt.Sprintf("las header read completed: found %d points", lasFile.NumberOfPoints()))
	emitEvent(EventReadCRSDetected, opts, start, inputDesc, fmt.Sprintf("crs: %s", lasFile.GetCRS()))

	// LOAD POINTS
	emitEvent(EventPointLoadingStarted, opts, start, inputDesc, "point loading started")
	mutatorPipeline := mutator.NewPipeline(opts.mutators...)
	err = tr.Load(lasFile, t.convFactory, mutatorPipeline, ctx)
	if err != nil {
		emitEvent(EventPointLoadingError, opts, start, inputDesc, fmt.Sprintf("load error: %v", err))
		return err
	}
	emitEvent(EventPointLoadingCompleted, opts, start, inputDesc, "point loading completed")

	// BUILD TREE
	emitEvent(EventBuildStarted, opts, start, inputDesc, "build started")
	err = tr.Build()
	if err != nil {
		emitEvent(EventBuildError, opts, start, inputDesc, fmt.Sprintf("build error: %v", err))
		return err
	}
	emitEvent(EventBuildCompleted, opts, start, inputDesc, "build completed")

	// EXPORT
	emitEvent(EventExportStarted, opts, start, inputDesc, "export started")
	w, err := t.writerProvider(outputFolder, opts)
	if err != nil {
		emitEvent(EventBuildError, opts, start, inputDesc, fmt.Sprintf("export init error: %v", err))
		return err
	}
	err = w.Write(tr, "", ctx)
	if err != nil {
		emitEvent(EventBuildError, opts, start, inputDesc, fmt.Sprintf("export error: %v", err))
		return err
	}
	emitEvent(EventExportStarted, opts, start, inputDesc, fmt.Sprintf("export completed in %v seconds", time.Since(start).String()))
	return nil
}

func emitEvent(e TilerEvent, opts *TilerOptions, start time.Time, inputDesc string, msg string) {
	if opts.callback != nil {
		opts.callback(e, inputDesc, time.Since(start).Milliseconds(), msg)
	}
}
