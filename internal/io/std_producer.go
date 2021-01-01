package io

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"path"
	"strconv"
	"sync"
)

type StandardProducer struct {
	basePath string
	options  *tiler.TilerOptions
}

func NewStandardProducer(basepath string, subfolder string, options *tiler.TilerOptions) Producer {
	return &StandardProducer{
		basePath: path.Join(basepath, subfolder),
		options:  options,
	}
}

// Parses an octnode and submits WorkUnits the the provided workchannel. Should be called only on the tree root octNode.
// Closes the channel when all work is submitted.
func (p *StandardProducer) Produce(work chan *WorkUnit, wg *sync.WaitGroup, node octree.INode) {
	p.produce(p.basePath, node, work, wg)
	close(work)
	wg.Done()
}

// Parses an octnode and submits WorkUnits the the provided workchannel.
func (p *StandardProducer) produce(basePath string, node octree.INode, work chan *WorkUnit, wg *sync.WaitGroup) {
	// if node contains children (it should always be the case), then submit work
	if node.NumberOfPoints() > 0 {
		work <- &WorkUnit{
			OctNode:  node,
			BasePath: basePath,
			Opts:     p.options,
		}
	}

	// iterate all non nil children and recursively submit all work units
	for i, child := range node.GetChildren() {
		if child != nil && child.IsInitialized() {
			p.produce(path.Join(basePath, strconv.Itoa(i)), child, work, wg)
		}
	}
}
