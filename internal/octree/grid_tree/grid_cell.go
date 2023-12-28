package grid_tree

import (
	"math"

	"github.com/mfbonfigli/gocesiumtiler/internal/data"
)

// Data structure that accepts points and stores just the one closest to its center, or if the side is too small,
// all the points. It assumes that coordinates are expressed in a metric cartesian system.
type gridCell struct {
	points []*data.Point // points stored in the cell
}

// returns the spatial index component associated to a given dimension (e.g. X or Y or Z) coordinate value
func getDimensionIndex(dimensionValue float32, size float64) int {
	return int(math.Floor(float64(dimensionValue) / size))
}

// returns the cell center X,Y,Z coordinates from the spatial index of the cell and the cell size
func (gc *gridCell) getCellCenter(x, y, z int, size float64) (float64, float64, float64) {
	return float64(x)*size + size/2,
		float64(y)*size + size/2,
		float64(z)*size + size/2
}

// submits a point to the cell, eventually returning a pointer to the point pushed out.
func (gc *gridCell) pushPoint(point *data.Point, size, sizeThreshold float64, x, y, z int) *data.Point {
	if gc.points == nil {
		gc.storeFirstPoint(point)
		return nil
	}

	if size < sizeThreshold {
		gc.points = append(gc.points, point)
		return nil
	}

	return gc.storeClosestPointAndReturnFarthestOne(point, x, y, z, size)
}

// sets the points slice to a new slice containing the input point and stores its distanceFromCenter
func (gc *gridCell) storeFirstPoint(point *data.Point) {
	gc.points = []*data.Point{point}
}

// takes the input point and compares its distance from the center to the one in the points array,
// storing in the array only the one closest to the center and returning the other, rejected and farthest from the center, one
func (gc *gridCell) storeClosestPointAndReturnFarthestOne(point *data.Point, x int, y int, z int, size float64) *data.Point {
	distanceNew := gc.getDistanceFromCenter(point, x, y, z, size)
	distanceExisting := gc.getDistanceFromCenter(gc.points[0], x, y, z, size)

	if distanceNew < distanceExisting {
		oldPoint := gc.points[0]
		gc.points[0] = point
		return oldPoint
	}

	return point
}

// computes the cartesian distance of a point from the cell center
func (gc *gridCell) getDistanceFromCenter(point *data.Point, x int, y int, z int, size float64) float64 {
	xc, yc, zc := gc.getCellCenter(x, y, z, size)

	return math.Sqrt(
		math.Pow(float64(point.X)-xc, 2) +
			math.Pow(float64(point.Y)-yc, 2) +
			math.Pow(float64(point.Z)-zc, 2),
	)
}
