package gh_ellipsoid_to_geoid_z_converter

import (
	"github.com/mfbonfigli/gocesiumtiler/converters"
	"github.com/mfbonfigli/gocesiumtiler/structs/geometry"
)

type ghElevationConverter struct {
	GravitationalModel *egm
	CoordinateConverter converters.CoordinateConverter
}

func NewGHElevationConverter(coordinateConverter converters.CoordinateConverter) converters.EllipsoidToGeoidZConverter {
	var gravitationalModel = newDefaultEarthGravitationalModel()

	return &ghElevationConverter{
		GravitationalModel: gravitationalModel,
		CoordinateConverter: coordinateConverter,
	}
}

func (gHElevationConverter *ghElevationConverter) GetEllipsoidToGeoidZOffset(lat, lon float64, sourceSrid int) (float64, error) {
	coordinateInEPSG4326, err := gHElevationConverter.CoordinateConverter.ConvertCoordinateSrid(sourceSrid, 4326, geometry.Coordinate{X: &lat, Y: &lon, Z: nil})
	if err != nil {
		return 0, err
	}

	return gHElevationConverter.GravitationalModel.heightOffset(*coordinateInEPSG4326.X, *coordinateInEPSG4326.Y, 0), err
}
