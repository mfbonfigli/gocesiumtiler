package geoid_offset

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
)

// this geoid to ellipsoid converter caches the offset of the first point passed to it and
// returns it for all subsequent calls. It is very efficient but unsuitable for very large spatial
// point clouds (several km2).
type EllipsoidToGeoidSinglePointCalculator struct {
	cachedOffset                     *float64
	ellipsoidToGeoidOffsetCalculator converters.EllipsoidToGeoidOffsetCalculator
}

func NewEllipsoidToGeoidSinglePointCalculator(ellipsoidToGeoidOffsetCalculator converters.EllipsoidToGeoidOffsetCalculator) converters.EllipsoidToGeoidOffsetCalculator {
	return &EllipsoidToGeoidSinglePointCalculator{
		ellipsoidToGeoidOffsetCalculator: ellipsoidToGeoidOffsetCalculator,
	}
}

func (spc *EllipsoidToGeoidSinglePointCalculator) GetEllipsoidToGeoidOffset(lon, lat float64, srid int) (float64, error) {
	if spc.cachedOffset == nil {
		offset, err := spc.getEllipsoidToGeoidOffset(lon, lat, srid)
		if err != nil {
			return 0, nil
		}
		spc.cachedOffset = &offset
	}

	return *spc.cachedOffset, nil
}

func (spc *EllipsoidToGeoidSinglePointCalculator) getEllipsoidToGeoidOffset(lon, lat float64, srid int) (float64, error) {
	return spc.ellipsoidToGeoidOffsetCalculator.GetEllipsoidToGeoidOffset(lat, lon, srid)
}
