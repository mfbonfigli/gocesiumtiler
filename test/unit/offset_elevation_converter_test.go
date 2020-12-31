package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/offset_elevation_corrector"
	"testing"
)

func TestElevationIsAdded(t *testing.T) {
	expected := 10.68
	offsetElevationCorrector := offset_elevation_corrector.NewOffsetElevationCorrector(7.57)
	actual := offsetElevationCorrector.CorrectElevation(0, 0, 3.11)
	if actual != expected {
		t.Errorf("Expected Elevation = %f, got %f", expected, actual)
	}
}

func TestElevationIsSubtracted(t *testing.T) {
	expected := 3.0
	offsetElevationCorrector := offset_elevation_corrector.NewOffsetElevationCorrector(-0.11)
	actual := offsetElevationCorrector.CorrectElevation(0, 0, 3.11)
	if actual != expected {
		t.Errorf("Expected Elevation = %f, got %f", expected, actual)
	}
}

func TestElevationIsLeftUnchanged(t *testing.T) {
	expected := 3.11
	offsetElevationCorrector := offset_elevation_corrector.NewOffsetElevationCorrector(0)
	actual := offsetElevationCorrector.CorrectElevation(0, 0, 3.11)
	if actual != expected {
		t.Errorf("Expected Elevation = %f, got %f", expected, actual)
	}
}
