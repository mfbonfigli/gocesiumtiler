package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
)

// Writer writes a tree as a 3D Cesium Point cloud to the given output folder
type Writer interface {
	Write(t tree.Tree, folderName string, ctx context.Context) error
}

type StandardWriter struct {
	numWorkers   int
	bufferRatio  int
	basePath     string
	version      version.TilesetVersion
	squash       bool
	producerFunc func(basepath, folder string) Producer
	consumerFunc func(version.TilesetVersion) Consumer
}

func NewWriter(basePath string, options ...func(*StandardWriter)) (*StandardWriter, error) {
	w := &StandardWriter{
		basePath:     basePath,
		numWorkers:   1,
		bufferRatio:  5,
		version:      version.TilesetVersion_1_0,
		producerFunc: NewStandardProducer,
	}
	for _, optFn := range options {
		optFn(w)
	}

	// Set consumerFunc after options are applied so we can use the squash flag
	w.consumerFunc = func(v version.TilesetVersion) Consumer {
		if v == version.TilesetVersion_1_0 {
			return NewStandardConsumer(WithGeometryEncoder(NewPntsEncoder()), WithSkipTileset(w.squash))
		}
		return NewStandardConsumer(WithGeometryEncoder(NewGltfEncoder()), WithSkipTileset(w.squash))
	}

	return w, nil
}

// WithNumWorkers defines how many writer goroutines to launch when writing the tiles.
func WithNumWorkers(n int) func(*StandardWriter) {
	return func(w *StandardWriter) {
		w.numWorkers = n
	}
}

// WithBufferRation defines how many jobs per writer worker to allow enqueuing.
func WithBufferRatio(n int) func(*StandardWriter) {
	return func(w *StandardWriter) {
		w.bufferRatio = int(math.Max(1, float64(n)))
	}
}

// WithTilesetVersion sets the version of the generated tilesets. version 1.0 generates .pnts gemetries
// while version 1.1 generates .glb (gltf) geometries.
func WithTilesetVersion(v version.TilesetVersion) func(*StandardWriter) {
	return func(w *StandardWriter) {
		w.version = v
	}
}

// WithSquash sets whether to generate a single tileset.json file instead of multiple files
func WithSquash(squash bool) func(*StandardWriter) {
	return func(w *StandardWriter) {
		w.squash = squash
	}
}

func (w *StandardWriter) Write(t tree.Tree, folderName string, ctx context.Context) error {
	// init channel where consumers can eventually submit errors that prevented them to finish the job
	errorChannel := make(chan error)

	// init channel where to submit work with a buffer N times greater than the number of consumer
	workChannel := make(chan *WorkUnit, w.numWorkers*w.bufferRatio)

	var waitGroup sync.WaitGroup
	var errorWaitGroup sync.WaitGroup

	// producing is easy, only 1 producer
	producer := w.producerFunc(w.basePath, folderName)
	waitGroup.Add(1)
	go producer.Produce(workChannel, errorChannel, &waitGroup, t.RootNode(), ctx)

	// add consumers to waitgroup and launch them
	for i := 0; i < w.numWorkers; i++ {
		waitGroup.Add(1)
		// instantiate a new converter per each goroutine for thread safety
		consumer := w.consumerFunc(w.version)
		go consumer.Consume(workChannel, errorChannel, &waitGroup)
	}

	// launch error listener
	errs := []error{}
	errorWaitGroup.Add(1)
	go func() {
		defer errorWaitGroup.Done()
		for {
			err, ok := <-errorChannel
			if !ok {
				return
			}
			errs = append(errs, err)
		}
	}()

	// wait for producers and consumers to finish
	waitGroup.Wait()

	// close error chan
	close(errorChannel)
	errorWaitGroup.Wait()

	if len(errs) != 0 {
		return errs[0]
	}

	// If squash mode is enabled, generate the single tileset.json file
	if w.squash {
		return w.writeSquashedTileset(t, folderName)
	}

	return nil
}

// writeSquashedTileset generates a single tileset.json file with all nodes
func (w *StandardWriter) writeSquashedTileset(t tree.Tree, folderName string) error {
	rootTile := w.buildTileTree(t.RootNode(), "")

	tileset := Tileset{
		Asset: Asset{
			Version: w.version,
		},
		GeometricError: t.RootNode().GeometricError(),
		Root:           rootTile,
	}

	file, err := json.Marshal(tileset)
	if err != nil {
		return fmt.Errorf("failed to marshal tileset: %w", err)
	}

	return os.WriteFile(path.Join(w.basePath, folderName, "tileset.json"), file, 0644)
}

// buildTileTree builds the complete tile hierarchy for squashed mode
func (w *StandardWriter) buildTileTree(node tree.Node, parentPath string) Root {
	reg := node.BoundingBox()

	var cMajorTransformPtr *[16]float64
	if trans := node.ToParentCRS(); trans != nil && *trans != model.IdentityTransform {
		cMajor := trans.ForwardColumnMajor()
		cMajorTransformPtr = &cMajor
	}

	root := Root{
		BoundingVolume: BoundingVolume{Box: reg.AsCesiumBox()},
		GeometricError: node.GeometricError(),
		Refine:         "ADD",
		Transform:      cMajorTransformPtr,
	}

	// Add content if node has points
	if node.TotalNumberOfPoints() > 0 {
		filename := "content.pnts"
		if w.version == version.TilesetVersion_1_1 {
			filename = "content.glb"
		}
		if parentPath != "" {
			root.Content = &Content{Url: path.Join(parentPath, filename)}
		} else {
			root.Content = &Content{Url: filename}
		}
	}

	// Add children recursively
	for i, child := range node.Children() {
		if child != nil && child.TotalNumberOfPoints() > 0 {
			childPath := path.Join(parentPath, strconv.Itoa(i))
			childTile := w.buildChildTile(child, childPath)
			root.Children = append(root.Children, childTile)
		}
	}

	return root
}

// buildChildTile builds a child tile for squashed mode
func (w *StandardWriter) buildChildTile(node tree.Node, nodePath string) *Child {
	reg := node.BoundingBox()

	child := &Child{
		BoundingVolume: BoundingVolume{Box: reg.AsCesiumBox()},
		GeometricError: node.GeometricError(),
		Refine:         "ADD",
	}

	// Add content if node has points
	if node.TotalNumberOfPoints() > 0 {
		filename := "content.pnts"
		if w.version == version.TilesetVersion_1_1 {
			filename = "content.glb"
		}
		child.Content = &Content{Url: path.Join(nodePath, filename)}
	}

	// Add children recursively
	for i, childNode := range node.Children() {
		if childNode != nil && childNode.TotalNumberOfPoints() > 0 {
			childPath := path.Join(nodePath, strconv.Itoa(i))
			grandChild := w.buildChildTile(childNode, childPath)
			child.Children = append(child.Children, grandChild)
		}
	}

	return child
}
