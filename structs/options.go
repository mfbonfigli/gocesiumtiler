package structs

type TilerOptions struct {
	Input string
	Output string
	Srid int
	ColorDepth int
	ZOffset float64
	MaxNumPointsPerNode int32
	EnableGeoidZCorrection bool
}