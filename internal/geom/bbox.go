package geom

type BoundingBox struct {
	Xmin, Xmax, Ymin, Ymax, Zmin, Zmax float64
}

// Constructor to properly initialize a boundingBox struct computing the mids
func NewBoundingBox(Xmin, Xmax, Ymin, Ymax, Zmin, Zmax float64) BoundingBox {
	bbox := BoundingBox{
		Xmin: Xmin,
		Xmax: Xmax,
		Ymin: Ymin,
		Ymax: Ymax,
		Zmin: Zmin,
		Zmax: Zmax,
	}
	return bbox
}

// Computes a bounding box from the given box and the given octant index using custom split points
func NewBoundingBoxFromParentWithSplits(parent BoundingBox, octant int, splitX, splitY, splitZ float64) BoundingBox {
	var xMin, xMax, yMin, yMax, zMin, zMax float64
	switch octant {
	case 0, 2, 4, 6:
		xMin = parent.Xmin
		xMax = splitX
	case 1, 3, 5, 7:
		xMin = splitX
		xMax = parent.Xmax
	}
	switch octant {
	case 0, 1, 4, 5:
		yMin = parent.Ymin
		yMax = splitY
	case 2, 3, 6, 7:
		yMin = splitY
		yMax = parent.Ymax
	}
	switch octant {
	case 0, 1, 2, 3:
		zMin = parent.Zmin
		zMax = splitZ
	case 4, 5, 6, 7:
		zMin = splitZ
		zMax = parent.Zmax
	}
	return NewBoundingBox(xMin, xMax, yMin, yMax, zMin, zMax)
}

// AsCesiumBox returns the bounding box expressed according to the cesium "box" format
func (b BoundingBox) Center() (x, y, z float64) {
	return (b.Xmax + b.Xmin) / 2, (b.Ymax + b.Ymin) / 2, (b.Zmax + b.Zmin) / 2
}

// AsCesiumBox returns the bounding box expressed according to the cesium "box" format
func (b BoundingBox) AsCesiumBox() [12]float64 {
	xMid, yMid, zMid := b.Center()
	return [12]float64{
		xMid, yMid, zMid,
		(b.Xmax - xMid), 0, 0,
		0, (b.Ymax - yMid), 0,
		0, 0, (b.Zmax - zMid),
	}
}
