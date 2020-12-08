package io

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

// Parses an octnode and submits WorkUnits the the provided workchannel. Should be called only on the tree root octNode.
// Closes the channel when all work is submitted.
func Produce(basepath string, node octree.INode, opts *tiler.TilerOptions, work chan *WorkUnit, wg *sync.WaitGroup, subfolder string) {
	produce(filepath.Join(basepath, subfolder), node, opts, work, wg)
	close(work)
	wg.Done()
}

// Parses an octnode and submits WorkUnits the the provided workchannel.
func produce(basepath string, node octree.INode, opts *tiler.TilerOptions, work chan *WorkUnit, wg *sync.WaitGroup) {
	// if node contains children (it should always be the case), then submit work
	if node.GetLocalChildrenCount() > 0 {
		work <- &WorkUnit{
			OctNode:  node,
			BasePath: basepath,
			Opts:     opts,
		}
	}

	// iterate all non nil children and recursively submit all work units
	for i, child := range node.GetChildren() {
		if child != nil && child.IsInitialized() {
			produce(path.Join(basepath, strconv.Itoa(i)), child, opts, work, wg)
		}
	}
}
