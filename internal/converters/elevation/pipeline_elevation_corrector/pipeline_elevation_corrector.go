package pipeline_elevation_corrector

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
)

type PipelineElevationCorrector struct {
	correctors []converters.ElevationCorrector
}

func NewPipelineElevationCorrector(elevationCorrectors []converters.ElevationCorrector) converters.ElevationCorrector {
	return &PipelineElevationCorrector{
		correctors: elevationCorrectors,
	}
}

func (c *PipelineElevationCorrector) CorrectElevation(lon, lat, z float64) float64 {
	for _, elevationCorrector := range c.correctors {
		z += elevationCorrector.CorrectElevation(lon, lat, z)
	}

	return z
}
