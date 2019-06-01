package octree

type BoundingBox struct {
	Xmin, Xmax, Ymin, Ymax, Zmin, Zmax, Xmid, Ymid, Zmid float64
}

// Constructor to properly initialize a BoundingBox struct computing the mids
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

func newBoundingBox(parent *BoundingBox, octant *uint8) *BoundingBox {
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

func (bbox *BoundingBox) getVolume() float64 {
	return (bbox.Xmax - bbox.Xmin) * (bbox.Ymax - bbox.Ymin) * (bbox.Zmax - bbox.Zmin)
}

func (bbox *BoundingBox) getOctantFromElement(element *OctElement) uint8 {
	var result uint8 = 0
	if float64(element.X) > bbox.Xmid {
		result += 1
	}
	if float64(element.Y) > bbox.Ymid {
		result += 2
	}
	if float64(element.Z) > bbox.Zmid {
		result += 4
	}
	return result
}

func (bbox *BoundingBox) getOctantBoundingBox(octant *uint8) *BoundingBox {
	return newBoundingBox(bbox, octant)
}
