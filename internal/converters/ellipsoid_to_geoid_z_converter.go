package converters

type EllipsoidToGeoidZConverter interface {
	GetEllipsoidToGeoidZOffset(lat, lon float64, sourceSrid int) (float64, error)
}
