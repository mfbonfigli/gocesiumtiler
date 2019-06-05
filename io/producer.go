package io

import (
	"go_cesium_tiler/structs/octree"
	"path"
	"path/filepath"
	"strconv"
	"sync"
)

// Parses an octnode and submits WorkUnits the the provided workchannel. Should be called only on the tree root OctNode.
// Closes the channel when all work is submitted.
func Produce(basepath string, node *octree.OctNode, opts *octree.TilerOptions, work chan *WorkUnit, wg *sync.WaitGroup, subfolder string) {
	produce(filepath.Join(basepath, subfolder), node, opts, work, wg)
	close(work)
	wg.Done()
}

// Parses an octnode and submits WorkUnits the the provided workchannel.
func produce(basepath string, node *octree.OctNode, opts *octree.TilerOptions, work chan *WorkUnit, wg *sync.WaitGroup) {
	// if node contains children (it should always be the case), then submit work
	if node.LocalChildrenCount > 0 {
		work <- &WorkUnit{
			OctNode:  node,
			BasePath: basepath,
			Opts:     opts,
		}
	}

	// iterate all non nil children and recursively submit all work units
	for i, child := range node.Children {
		if child != nil && child.Initialized {
			produce(path.Join(basepath, strconv.Itoa(i)), child, opts, work, wg)
		}
	}
}
