package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/coordinate/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_offset/gh_offset_calculator"
	"math"
	"testing"
)

var offsetCalculator = gh_offset_calculator.NewEllipsoidToGeoidGHOffsetCalculator(proj4_coordinate_converter.NewProj4CoordinateConverter())

func TestGetEllipsoidToGeoidZOffsetFrom32633Correct(t *testing.T) {
	expected := 48.95
	output, err := offsetCalculator.GetEllipsoidToGeoidOffset(4576930.54, 491880.85, 32633)

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
	output, err := offsetCalculator.GetEllipsoidToGeoidOffset(41.343825, 14.902954, 4326)

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
