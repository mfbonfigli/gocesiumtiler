package geom

import (
	"testing"
)

func TestNewBBox(t *testing.T) {
	actual := NewBoundingBox(-10, 0, 10, 20, -100, 20)
	expected := BoundingBox{
		Xmin: -10,
		Xmax: 0,
		Ymin: 10,
		Ymax: 20,
		Zmin: -100,
		Zmax: 20,
	}
	if actual != expected {
		t.Errorf("expected boundingbox %v got %v", expected, actual)
	}
}

func TestBBoxAsCesiumBox(t *testing.T) {
	b := NewBoundingBox(-30, 10, 10, 20, -100, 20)
	expected := [12]float64{
		-10, 15, -40,
		20, 0, 0,
		0, 5, 0,
		0, 0, 60,
	}

	if actual := b.AsCesiumBox(); actual != expected {
		t.Errorf("expected boundingbox array %v got %v", expected, actual)
	}
}
