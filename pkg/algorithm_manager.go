package pkg

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_elevation_corrector"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/offset_elevation_corrector"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree/random_trees"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"log"
)

type IAlgorithmManager interface {
	GetElevationCorrectionAlgorithm(opts *tiler.TilerOptions) converters.ElevationCorrector
	GetTreeAlgorithm(options *tiler.TilerOptions, corrector converters.ElevationCorrector) octree.ITree
}

type AlgorithmManager struct{}

func NewAlgorithmManager() IAlgorithmManager {
	return &AlgorithmManager{}
}

func (algorithmManager *AlgorithmManager) GetElevationCorrectionAlgorithm(opts *tiler.TilerOptions) converters.ElevationCorrector {
	if !opts.EnableGeoidZCorrection {
		return offset_elevation_corrector.NewOffsetElevationCorrector(opts.ZOffset)
	} else {
		return geoid_elevation_corrector.NewGeoidElevationCorrector(opts.ZOffset, opts.ElevationConverter)
	}
}

func (algorithmManager *AlgorithmManager) GetTreeAlgorithm(options *tiler.TilerOptions, corrector converters.ElevationCorrector) octree.ITree {
	switch options.Strategy {
	case tiler.BoxedRandom:
		return random_trees.NewBoxedRandomTree(options, options.CoordinateConverter, corrector)
	case tiler.FullyRandom:
		return random_trees.NewRandomTree(options, options.CoordinateConverter, corrector)
	}

	log.Fatal("Unrecognized strategy")
	return nil
}
