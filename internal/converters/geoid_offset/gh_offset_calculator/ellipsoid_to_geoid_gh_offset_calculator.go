package gh_offset_calculator

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
)

type EllipsoidToGeoidGHOffsetCalculator struct {
	gravitationalModel  *egm
	coordinateConverter converters.CoordinateConverter
}

func NewEllipsoidToGeoidGHOffsetCalculator(coordinateConverter converters.CoordinateConverter) converters.EllipsoidToGeoidOffsetCalculator {
	var gravitationalModel = newDefaultEarthGravitationalModel()

	return &EllipsoidToGeoidGHOffsetCalculator{
		gravitationalModel:  gravitationalModel,
		coordinateConverter: coordinateConverter,
	}
}

func (ghc *EllipsoidToGeoidGHOffsetCalculator) GetEllipsoidToGeoidOffset(lat, lon float64, sourceSrid int) (float64, error) {
	coordinateInEPSG4326, err := ghc.coordinateConverter.ConvertCoordinateSrid(sourceSrid, 4326, geometry.Coordinate{X: &lon, Y: &lat, Z: nil})
	if err != nil {
		return 0, err
	}

	return ghc.gravitationalModel.heightOffset(*coordinateInEPSG4326.X, *coordinateInEPSG4326.Y, 0), err
}
