package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/elevation/pipeline_elevation_corrector"
	"testing"
)

type mockElevationCorrector struct{
}

func (m *mockElevationCorrector) CorrectElevation(lon, lat, z float64) float64 {
	return z * 2
}

func TestElevationCorrectionsAreSummed(t *testing.T) {
	expected := 4.8
	var correctors = []converters.ElevationCorrector{
		&mockElevationCorrector{},
		&mockElevationCorrector{},
	}

	pipelineCorrector := pipeline_elevation_corrector.NewPipelineElevationCorrector(correctors)

	actual := pipelineCorrector.CorrectElevation(14, 41, 1.2)

	if actual != expected {
		t.Errorf("Expected Elevation = %f, got %f", expected, actual)
	}
}
