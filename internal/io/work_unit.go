package io

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
)

// Contains the minimal data needed to produce a single 3d tile, i.e. a binary content.pnts file and a tileset.json file
type WorkUnit struct {
	Node     octree.INode
	Opts     *tiler.TilerOptions
	BasePath string
}
