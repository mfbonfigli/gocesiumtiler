package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/elevation/geoid_elevation_corrector"
	"math"
	"testing"
)

type mockOffsetCalculator struct{}

func (mockOffsetCalculator *mockOffsetCalculator) GetEllipsoidToGeoidOffset(lat, lon float64, sourceSrid int) (float64, error) {
	return 48.95, nil
}

func TestGeoidElevationCorrector(t *testing.T) {
	var bufferedElevationConverter = geoid_elevation_corrector.NewGeoidElevationCorrector(
		32633,
		&mockOffsetCalculator{}, // using a mock to avoid dependencies on other classes
	)

	expected := 58.95
	output := bufferedElevationConverter.CorrectElevation(491880.85, 4576930.54, 10.0)

	if math.Abs(expected-output) > 1E-3 {
		t.Errorf(
			"Expected X:%.3f, got X:%.3f",
			expected,
			output,
		)
	}
}
