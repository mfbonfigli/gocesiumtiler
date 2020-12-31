package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/gh_ellipsoid_to_geoid_z_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/proj4_coordinate_converter"
	"math"
	"testing"
)

func TestBufferedGetEllipsoidToGeoidZOffsetFrom32633Correct(t *testing.T) {
	var bufferedElevationConverter = converters.NewElevationConverterBuffer(
		32633,
		360/(6371000*math.Pi*2),
		gh_ellipsoid_to_geoid_z_converter.NewGHElevationConverter(proj4_coordinate_converter.NewProj4CoordinateConverter()),
	)

	expected := 58.95
	output, err := bufferedElevationConverter.GetConvertedElevation(491880.85, 4576930.54, 10)

	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err.Error())
	}

	if math.Abs(expected-output) > 1E-3 {
		t.Errorf(
			"Expected X:%.3f, got X:%.3f",
			expected,
			output,
		)
	}
}
