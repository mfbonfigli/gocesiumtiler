package geoid_elevation_corrector

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/geoid_offset"
	"log"
)

type GeoidElevationCorrector struct {
	srid             int
	offsetCalculator converters.EllipsoidToGeoidOffsetCalculator
}

func NewGeoidElevationCorrector(srid int, ellipsoidToGeoidOffsetCalculator converters.EllipsoidToGeoidOffsetCalculator) converters.ElevationCorrector {
	// TODO: by default we are using the ellipsoidToGeoidSinglePointConverter as the old buffered converter
	//  suffers of coupling problems with the srid of data. It needs a cell size but this depends on the SRID and
	//  thus either a way to dynamically estabilish according to the srid is found or we can only use it if data is in 4326 srid
	//  which introduces an undocumented requirement. We need to fix the EllipsoidToGeoidBufferedCalculator to allow it
	//  to be used here
	return &GeoidElevationCorrector{
		srid:             srid,
		offsetCalculator: geoid_offset.NewEllipsoidToGeoidSinglePointCalculator(ellipsoidToGeoidOffsetCalculator),
	}
}

func (c *GeoidElevationCorrector) CorrectElevation(lon, lat, z float64) float64 {
	zfix, err := c.offsetCalculator.GetEllipsoidToGeoidOffset(lon, lat, c.srid)
	if err != nil {
		log.Fatal(err)
	}
	return zfix + z
}
