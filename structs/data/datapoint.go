package data

// Contains data of a Point Cloud Point, namely X,Y,Z coords,
// R,G,B color components, Intensity and Classification
type Point struct {
	X              float64
	Y              float64
	Z              float64
	R              uint8
	G              uint8
	B              uint8
	Intensity      uint8
	Classification uint8
}

// Builds a new Point from the given coordinates, colors, intensity and classification values
func NewPoint(X, Y, Z float64, R, G, B, Intensity, Classification uint8) *Point {
	return &Point{
		X:              X,
		Y:              Y,
		Z:              Z,
		R:              R,
		G:              G,
		B:              B,
		Intensity:      Intensity,
		Classification: Classification,
	}
}
