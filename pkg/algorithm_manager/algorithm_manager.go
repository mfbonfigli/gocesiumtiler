package algorithm_manager

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
)

type AlgorithmManager interface {
	GetElevationCorrectionAlgorithm() converters.ElevationCorrector
	GetTreeAlgorithm() octree.ITree
	GetCoordinateConverterAlgorithm() converters.CoordinateConverter
}
