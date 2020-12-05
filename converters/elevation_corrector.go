package converters

type ElevationCorrector interface {
	CorrectElevation(lon, lat, z float64) float64
}
