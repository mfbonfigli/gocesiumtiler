package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/gh_ellipsoid_to_geoid_z_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/proj4_coordinate_converter"
	"math"
	"testing"
)

var elevationConverter = gh_ellipsoid_to_geoid_z_converter.NewGHElevationConverter(proj4_coordinate_converter.NewProj4CoordinateConverter())

func TestGetEllipsoidToGeoidZOffsetFrom32633Correct(t *testing.T) {
	expected := 48.95
	output, err := elevationConverter.GetEllipsoidToGeoidZOffset(491880.85, 4576930.54, 32633)

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

func TestGetEllipsoidToGeoidZOffsetFrom4326Correct(t *testing.T) {
	expected := 48.95
	output, err := elevationConverter.GetEllipsoidToGeoidZOffset(14.902954, 41.343825, 4326)

	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err.Error())
	}

	if math.Abs(expected-output) > 1E-3 {
		t.Errorf(
			"Expected X:%.3f, got X:%.3f",
			expected,
			math.Round(output*1E6)/1E6,
		)
	}
}
