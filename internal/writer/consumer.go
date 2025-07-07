package writer

import (
	"fmt"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
)

// GeometryEncoder encodes a tree.Node into a binary file, like a .pnts or .glb/.gltf files.
type GeometryEncoder interface {
	Write(n tree.Node, folderPath string, prefix string) error
	TilesetVersion() version.TilesetVersion
}

type Consumer interface {
	Consume(workchan chan *WorkUnit, errchan chan error, waitGroup *sync.WaitGroup)
}

type StandardConsumer struct {
	encoder GeometryEncoder
}

func NewStandardConsumer(optFn ...func(*StandardConsumer)) Consumer {
	c := &StandardConsumer{
		encoder: NewPntsEncoder("d.pnts"),
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

	// encodes and writes the geometries to the disk as a .pnts/.glb file
	err := c.encoder.Write(node, parentFolder, workUnit.Prefix)
	if err != nil {
		return err
	}
	return nil
}
