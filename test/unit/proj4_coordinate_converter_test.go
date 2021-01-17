package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/coordinate/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"math"
	"testing"
)

var coordinateConverter = proj4_coordinate_converter.NewProj4CoordinateConverter()

func TestConvertsCoordinate(t *testing.T) {
	var testData = []struct {
		X           float64
		Y           float64
		Z           float64
		xExpected   float64
		yExpected   float64
		zExpected   float64
		inEpsgCode  int
		outEpsgCode int
		toleranceXY float64
		toleranceZ  float64
	}{
		{491880.85, 4576930.54, 10.0, 14.902954, 41.343825, 10.0, 32633, 4326, 5E-7, 0.0},
		{-121.1392808, 44.6588803, 10.0, -121.140535, 44.658728, 10.0, 4267, 4326, 5E-7, 0.0},
		{504544.56, 4848085.02, 10.0, -116.9435192, 43.7858081, 10.0, 2955, 4326, 5E-7, 0.0},
		{504544.56, 4848085.02, 0.0, -2089741.61, -4111363.09, 4390941.2622281, 2955, 4978, 5E-3, 5E-3},
	}

	for _, data := range testData {
		output, err := coordinateConverter.ConvertCoordinateSrid(
			data.inEpsgCode,
			data.outEpsgCode,
			geometry.Coordinate{
				X: data.X,
				Y: data.Y,
				Z: data.Z,
			},
		)

		if err != nil {
			t.Errorf("Unexpected error occurred: %s", err.Error())
		}

		if math.Abs(output.X-data.xExpected) > data.toleranceXY {
			t.Errorf(
				"Expected X within %.8f ± %.8f, got X:%.8f",
				data.xExpected,
				data.toleranceXY,
				output.X,
			)
		}

		if math.Abs(output.Y-data.yExpected) > data.toleranceXY {
			t.Errorf(
				"Expected Y within %.8f ± %.8f, got Y:%.8f",
				data.yExpected,
				data.toleranceXY,
				output.Y,
			)
		}

		if math.Abs(output.Z-data.zExpected) > data.toleranceZ {
			t.Errorf(
				"Expected Z within %.8f ± %.8f, got Z:%.8f",
				data.zExpected,
				data.toleranceZ,
				output.Z,
			)
		}
	}
}

func TestConvertsFromUnknownSridReturnsError(t *testing.T) {
	x := 491880.85
	y := 4576930.54
	z := 10.0

	_, err := coordinateConverter.ConvertCoordinateSrid(
		-1,
		4326,
		geometry.Coordinate{
			X: x,
			Y: y,
			Z: z,
		},
	)

	if err == nil {
		t.Errorf("Error was expected but none was returned")
	}
}

func TestConvertsToUnknownSridReturnsError(t *testing.T) {
	x := 491880.85
	y := 4576930.54
	z := 10.0

	_, err := coordinateConverter.ConvertCoordinateSrid(
		32633,
		-2,
		geometry.Coordinate{
			X: x,
			Y: y,
			Z: z,
		},
	)

	if err == nil {
		t.Errorf("Error was expected but none was returned")
	}
}

func TestConvertsFrom4326toWGS84Cartesian(t *testing.T) {
	x := 15.309277
	y := 41.363327
	z := 0.0

	xExpected := 4623905.13
	yExpected := 1265762.04
	zExpected := 4192791.72

	output, err := coordinateConverter.ConvertToWGS84Cartesian(
		geometry.Coordinate{
			X: x,
			Y: y,
			Z: z,
		},
		4326,
	)

	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err.Error())
	}

	if math.Abs(output.X-xExpected) > 5E-3 {
		t.Errorf(
			"Expected X: %.2f, got X:%.2f",
			xExpected,
			output.X,
		)
	}

	if math.Abs(output.Y-yExpected) > 5E-3 {
		t.Errorf(
			"Expected Y:%.2f, got Y:%.2f",
			yExpected,
			output.Y,
		)
	}

	if math.Abs(output.Z-zExpected) > 5E-3 {
		t.Errorf(
			"Expected Z:%.2f, got Z:%.2f",
			zExpected,
			output.Z,
		)
	}
}

func TestConvert326322DBoundingboxToWGS84Region(t *testing.T) {
	bbox := geometry.NewBoundingBox(
		430936.93,
		430946.93,
		4978549.23,
		4978559.23,
		0.0,
		10.0,
	)

	xMinExpected := 0.14179735
	xMaxExpected := 0.14179954
	yMinExpected := 0.78464809
	yMaxExpected := 0.78464968
	zMinExpected := 0.0
	zMaxExpected := 10.0

	boundingBoxOutput, err := coordinateConverter.Convert2DBoundingboxToWGS84Region(
		bbox,
		32632,
	)

	output := boundingBoxOutput.GetAsArray()

	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err.Error())
	}

	if math.Abs(output[0]-xMinExpected) > 1E-8 {
		t.Errorf(
			"Expected X min:%.8f, got X min:%.8f",
			xMinExpected,
			output[0],
		)
	}

	if math.Abs(output[1]-yMinExpected) > 1E-8 {
		t.Errorf(
			"Expected Y min:%.8f, got Y min:%.8f",
			yMinExpected,
			output[1],
		)
	}

	if math.Abs(output[4]-zMinExpected) > 1E-8 {
		t.Errorf(
			"Expected Z min:%.8f, got Z min:%.8f",
			zMinExpected,
			output[4],
		)
	}

	if math.Abs(output[2]-xMaxExpected) > 1E-8 {
		t.Errorf(
			"Expected X max:%.8f, got X max:%.8f",
			xMaxExpected,
			output[2],
		)
	}

	if math.Abs(output[3]-yMaxExpected) > 1E-8 {
		t.Errorf(
			"Expected Y max:%.8f, got Y max:%.8f",
			yMaxExpected,
			output[3],
		)
	}

	if math.Abs(output[5]-zMaxExpected) > 1E-8 {
		t.Errorf(
			"Expected Z max:%.8f, got Z max:%.8f",
			zMaxExpected,
			output[5],
		)
	}
}
