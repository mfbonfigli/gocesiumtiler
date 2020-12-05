package geoid_elevation_corrector

import (
	"log"
	"math"
	"github.com/mfbonfigli/gocesiumtiler/converters"
	"github.com/mfbonfigli/gocesiumtiler/converters/offset_elevation_corrector"
)

type GeoidElevationCorrector struct {
	offsetElevationCorrector converters.ElevationCorrector
	elevationConverterBuffer *converters.EllipsoidToGeoidBufferedConverter
}

func NewGeoidElevationCorrector(offset float64, elevationConverter converters.EllipsoidToGeoidZConverter) converters.ElevationCorrector {
	var offsetElevationCorrector = offset_elevation_corrector.NewOffsetElevationCorrector(offset)
	return &GeoidElevationCorrector{
		offsetElevationCorrector: offsetElevationCorrector,
		elevationConverterBuffer: converters.NewElevationConverterBuffer(4326, 360/6371000*math.Pi*2, elevationConverter),
	}
}

func (geoidElevationCorrector *GeoidElevationCorrector) CorrectElevation(lon, lat, z float64) float64 {
	zfix, err := geoidElevationCorrector.elevationConverterBuffer.GetConvertedElevation(lat, lon, z)
	if err != nil {
		log.Fatal(err)
	}
	return geoidElevationCorrector.offsetElevationCorrector.CorrectElevation(lon, lat, zfix)
}
