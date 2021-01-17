package converters

type EllipsoidToGeoidOffsetCalculator interface {
	GetEllipsoidToGeoidOffset(lat, lon float64, sourceSrid int) (float64, error)
}
