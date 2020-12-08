package geometry

import (
	"math"
)

const toRadians = math.Pi / 180
const toDeg = 180 / math.Pi

type BoundingBox struct {
	Xmin, Xmax, Ymin, Ymax, Zmin, Zmax, Xmid, Ymid, Zmid float64
}

// Constructor to properly initialize a boundingBox struct computing the mids
func NewBoundingBox(Xmin, Xmax, Ymin, Ymax, Zmin, Zmax float64) *BoundingBox {
	bbox := BoundingBox{
		Xmin: Xmin,
		Xmax: Xmax,
		Ymin: Ymin,
		Ymax: Ymax,
		Zmin: Zmin,
		Zmax: Zmax,
		Xmid: (Xmin + Xmax) / 2,
		Ymid: (Ymin + Ymax) / 2,
		Zmid: (Zmin + Zmax) / 2,
	}
	return &bbox
}

// Computes a bounding box from the given box and the given octant index
func NewBoundingBoxFromParent(parent *BoundingBox, octant *uint8) *BoundingBox {
	var xMin, xMax, yMin, yMax, zMin, zMax float64
	switch *octant {
	case 0, 2, 4, 6:
		xMin = parent.Xmin
		xMax = parent.Xmid
	case 1, 3, 5, 7:
		xMin = parent.Xmid
		xMax = parent.Xmax
	}
	switch *octant {
	case 0, 1, 4, 5:
		yMin = parent.Ymin
		yMax = parent.Ymid
	case 2, 3, 6, 7:
		yMin = parent.Ymid
		yMax = parent.Ymax
	}
	switch *octant {
	case 0, 1, 2, 3:
		zMin = parent.Zmin
		zMax = parent.Zmid
	case 4, 5, 6, 7:
		zMin = parent.Zmid
		zMax = parent.Zmax
	}
	return NewBoundingBox(xMin, xMax, yMin, yMax, zMin, zMax)
}

// Returns the approximate volume of the given bounding box, assuming that it is storing EPSG:4978 coordinates
func (bbox *BoundingBox) GetVolume() float64 {
	b := bbox.distance(bbox.Xmin, bbox.Xmax, bbox.Ymin, bbox.Ymin, 0, 0)
	h := bbox.distance(bbox.Xmin, bbox.Xmin, bbox.Ymin, bbox.Ymax, 0, 0)
	e := bbox.Zmax - bbox.Zmin
	return b * h * e
	//return (bbox.Xmax - bbox.Xmin) * (bbox.Ymax - bbox.Ymin) * (bbox.Zmax - bbox.Zmin)
}

func (bbox *BoundingBox) distance(lat1, lat2, lon1, lon2, el1, el2 float64) float64 {
	R := 6378137 / 1000; // Radius of the earth
	latDistance := (lat2 - lat1) * toRadians
	lonDistance := (lon2 - lon1) * toRadians
	a := math.Sin(latDistance/2)*math.Sin(latDistance/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(lonDistance/2)*math.Sin(lonDistance/2);
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := float64(R) * c * 1000 // convert to meters
	height := el1 - el2
	distance = distance*distance + height*height
	return math.Sqrt(distance)
}