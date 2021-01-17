package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/coordinate/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_offset"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_offset/gh_offset_calculator"
	"math"
	"testing"
)

func TestBufferedElevationConverter(t *testing.T) {
	var bufferedElevationConverter = geoid_offset.NewEllipsoidToGeoidBufferedCalculator(
		360/(6371000*math.Pi*2),
		gh_offset_calculator.NewEllipsoidToGeoidGHOffsetCalculator(proj4_coordinate_converter.NewProj4CoordinateConverter()),
	)
	expected := 48.95
	output, err := bufferedElevationConverter.GetEllipsoidToGeoidOffset(491880.85, 4576930.54, 32633)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	if math.Abs(expected-output) > 1E-3 {
		t.Errorf(
			"Expected X:%.3f, got X:%.3f",
			expected,
			output,
		)
	}
}
