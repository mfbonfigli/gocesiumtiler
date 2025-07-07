package writer

import (
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
)

// WorkUnit contains the minimal data needed to produce a single 3d tile, i.e.
// a binary content.pnts file, a tileset.json file
type WorkUnit struct {
	// Node contains the data for the current tile
	Node tree.Node
	// BasePath is the path of the folder where to write the content.pnts and tileset.json files for this workunit
	BasePath string
	// Prefix is the content file prefix
	Prefix string
}
