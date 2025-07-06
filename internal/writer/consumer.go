package writer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
)

// GeometryEncoder encodes a tree.Node into a binary file, like a .pnts or .glb/.gltf files.
type GeometryEncoder interface {
	Write(n tree.Node, folderPath string) error
	TilesetVersion() version.TilesetVersion
	Filename() string
}

type Consumer interface {
	Consume(workchan chan *WorkUnit, errchan chan error, waitGroup *sync.WaitGroup)
}

type StandardConsumer struct {
	encoder     GeometryEncoder
	skipTileset bool
}

func NewStandardConsumer(optFn ...func(*StandardConsumer)) Consumer {
	c := &StandardConsumer{
		encoder: NewPntsEncoder(),
	}
	for _, fn := range optFn {
		fn(c)
	}
	return c
}

// WithGeometryEncoder sets the consumer geometry encoder to the given one
func WithGeometryEncoder(e GeometryEncoder) func(*StandardConsumer) {
	return func(c *StandardConsumer) {
		c.encoder = e
	}
}

// WithSkipTileset sets whether to skip generating tileset.json files
// useful for example if the tileset is generated externally (squash)
func WithSkipTileset(skip bool) func(*StandardConsumer) {
	return func(c *StandardConsumer) {
		c.skipTileset = skip
	}
}

// Continually consumes WorkUnits submitted to a work channel producing corresponding gometry .pnts/.glb files and tileset.json files
// continues working until work channel is closed or if an error is raised. In this last case submits the error to an error
// channel before quitting
func (c *StandardConsumer) Consume(workchan chan *WorkUnit, errchan chan error, waitGroup *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			errchan <- fmt.Errorf("panic: %v", r)
		}
	}()
	// signal waitgroup finished work
	defer waitGroup.Done()
	for {
		// get work from channel
		work, ok := <-workchan
		if !ok {
			// channel was closed by producer, quit infinite loop
			break
		}

		// do work
		err := c.doWork(work)

		// if there were errors during work send in error channel and quit
		if err != nil {
			errchan <- err
			break
		}
	}

}

// Takes a workunit and writes the corresponding content.glb/.pnts and tileset.json files
func (c *StandardConsumer) doWork(workUnit *WorkUnit) error {
	parentFolder := workUnit.BasePath
	node := workUnit.Node

	// Create base folder if it does not exist
	err := utils.CreateDirectoryIfDoesNotExist(parentFolder)
	if err != nil {
		return err
	}
	// encodes and writes the geometries to the disk as a .pnts/.glb file
	err = c.encoder.Write(node, parentFolder)
	if err != nil {
		return err
	}
	// as an edge case we could have a leaf root node. This needs a tileset.json even if it's leaf.
	if !c.skipTileset && (!workUnit.Node.IsLeaf() || workUnit.Node.IsRoot()) {
		// if the node has children also writes the tileset.json file
		err := c.writeTilesetJsonFile(*workUnit)
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes the tileset.json file for the given WorkUnit
func (c *StandardConsumer) writeTilesetJsonFile(workUnit WorkUnit) error {
	parentFolder := workUnit.BasePath
	node := workUnit.Node

	// Create base folder if it does not exist
	err := utils.CreateDirectoryIfDoesNotExist(parentFolder)
	if err != nil {
		return err
	}

	// tileset.json file
	file := path.Join(parentFolder, "tileset.json")
	jsonData, err := c.generateTilesetJson(node)
	if err != nil {
		return err
	}

	// Writes the tileset.json binary content to the given file
	err = os.WriteFile(file, jsonData, 0666)
	if err != nil {
		return err
	}

	return nil
}

// Generates the tileset.json content for the given tree node
func (c *StandardConsumer) generateTilesetJson(node tree.Node) ([]byte, error) {
	if !node.IsLeaf() || node.IsRoot() {
		root, err := c.generateTilesetRoot(node)
		if err != nil {
			return nil, err
		}

		tileset := c.generateTileset(node, root)

		// Outputting a formatted json file
		e, err := json.Marshal(tileset)
		if err != nil {
			return nil, err
		}

		return e, nil
	}

	return nil, errors.New("this node is a non-root leaf, cannot create a tileset json for it")
}

func (c *StandardConsumer) generateTilesetRoot(node tree.Node) (Root, error) {
	reg := node.BoundingBox()

	children, err := c.generateTilesetChildren(node)
	if err != nil {
		return Root{}, err
	}

	var cMajorTransformPtr *[16]float64
	if trans := node.ToParentCRS(); trans != nil && *trans != model.IdentityTransform {
		cMajor := trans.ForwardColumnMajor()
		cMajorTransformPtr = &cMajor
	}

	return Root{
		Content:        &Content{c.encoder.Filename()},
		BoundingVolume: BoundingVolume{Box: reg.AsCesiumBox()},
		GeometricError: node.GeometricError(),
		Refine:         "ADD",
		Children:       children,
		Transform:      cMajorTransformPtr,
	}, nil
}

func (c *StandardConsumer) generateTileset(node tree.Node, root Root) Tileset {
	tileset := Tileset{}
	tileset.Asset = Asset{Version: c.encoder.TilesetVersion()}
	tileset.GeometricError = node.GeometricError()
	tileset.Root = root

	return tileset
}

func (c *StandardConsumer) generateTilesetChildren(node tree.Node) ([]*Child, error) {
	var children []*Child
	for i, child := range node.Children() {
		if c.nodeContainsPoints(child) {
			childTile := c.generateTilesetChild(child, i)
			children = append(children, &childTile)
		}
	}
	return children, nil
}

func (c *StandardConsumer) nodeContainsPoints(node tree.Node) bool {
	return node != nil && node.TotalNumberOfPoints() > 0
}

func (c *StandardConsumer) generateTilesetChild(child tree.Node, childIndex int) Child {
	childJson := Child{}
	filename := "tileset.json"
	if child.IsLeaf() {
		filename = c.encoder.Filename()
	}
	childJson.Content = &Content{
		Url: strconv.Itoa(childIndex) + "/" + filename,
	}
	reg := child.BoundingBox()
	childJson.BoundingVolume = BoundingVolume{
		Box: reg.AsCesiumBox(),
	}
	childJson.GeometricError = child.GeometricError()
	childJson.Refine = "ADD"
	return childJson
}
