package unit_test

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"math"
	"testing"
)

func TestBoundingBox(t *testing.T) {
	boundingBox := geometry.NewBoundingBox(-1.0, 3.0, -5.0, 5.0, -9.0, 7.0)

	if boundingBox.Xmin != -1.0 {
		t.Errorf("Expected Xmin:%f, got Xmin:%f", -1.0, boundingBox.Xmin)
	}
	if boundingBox.Xmid != 1.0 {
		t.Errorf("Expected Xmid:%f, got Xmid:%f", 1.0, boundingBox.Xmid)
	}
	if boundingBox.Xmax != 3.0 {
		t.Errorf("Expected Xmax:%f, got Xmax:%f", 3.0, boundingBox.Xmax)
	}
	if boundingBox.Ymin != -5.0 {
		t.Errorf("Expected Ymin:%f, got Ymin:%f", -5.0, boundingBox.Ymin)
	}
	if boundingBox.Ymid != 0.0 {
		t.Errorf("Expected Ymid:%f, got Ymid:%f", 0.0, boundingBox.Ymid)
	}
	if boundingBox.Ymax != 5.0 {
		t.Errorf("Expected Ymax:%f, got Ymax:%f", 5.0, boundingBox.Ymax)
	}
	if boundingBox.Zmin != -9.0 {
		t.Errorf("Expected Zmin:%f, got Zmin:%f", -9.0, boundingBox.Zmin)
	}
	if boundingBox.Zmid != -1.0 {
		t.Errorf("Expected Ymid:%f, got Ymid:%f", -1.0, boundingBox.Ymid)
	}
	if boundingBox.Zmax != 7.0 {
		t.Errorf("Expected Ymax:%f, got Ymax:%f", 5.0, boundingBox.Ymax)
	}
}

func TestBoundingBoxFromParent(t *testing.T) {
	parentBox := geometry.NewBoundingBox(-4.0, 4.0, -4.0, 4.0, -4.0, 4.0)
	testData := []struct {
		octant uint8
		Xmin   float64
		Xmid   float64
		Xmax   float64
		Ymin   float64
		Ymid   float64
		Ymax   float64
		Zmin   float64
		Zmid   float64
		Zmax   float64
	}{
		{0, -4.0, -2.0, 0.0, -4.0, -2.0, 0.0, -4.0, -2.0, 0.0},
		{1, 0.0, 2.0, 4.0, -4.0, -2.0, 0.0, -4.0, -2.0, 0.0},
		{2, -4.0, -2.0, 0.0, 0.0, 2.0, 4.0, -4.0, -2.0, 0.0},
		{3, 0.0, 2.0, 4.0, 0.0, 2.0, 4.0, -4.0, -2.0, 0.0},
		{4, -4.0, -2.0, 0.0, -4.0, -2.0, 0.0, 0.0, 2.0, 4.0},
		{5, 0.0, 2.0, 4.0, -4.0, -2.0, 0.0, 0.0, 2.0, 4.0},
		{6, -4.0, -2.0, 0.0, 0.0, 2.0, 4.0, 0.0, 2.0, 4.0},
		{7, 0.0, 2.0, 4.0, 0.0, 2.0, 4.0, 0.0, 2.0, 4.0},
	}

	for _, data := range testData {
		boundingBox := geometry.NewBoundingBoxFromParent(parentBox, &data.octant)

		if boundingBox.Xmin != data.Xmin {
			t.Errorf("Expected Xmin:%f, got Xmin:%f", data.Xmin, boundingBox.Xmin)
		}
		if boundingBox.Xmid != data.Xmid {
			t.Errorf("Expected Xmid:%f, got Xmid:%f", data.Xmid, boundingBox.Xmid)
		}
		if boundingBox.Xmax != data.Xmax {
			t.Errorf("Expected Xmax:%f, got Xmax:%f", data.Xmax, boundingBox.Xmax)
		}
		if boundingBox.Ymin != data.Ymin {
			t.Errorf("Expected Ymin:%f, got Ymin:%f", data.Ymin, boundingBox.Ymin)
		}
		if boundingBox.Ymid != data.Ymid {
			t.Errorf("Expected Ymid:%f, got Ymid:%f", data.Ymid, boundingBox.Ymid)
		}
		if boundingBox.Ymax != data.Ymax {
			t.Errorf("Expected Ymax:%f, got Ymax:%f", data.Ymax, boundingBox.Ymax)
		}
		if boundingBox.Zmin != data.Zmin {
			t.Errorf("Expected Zmin:%f, got Zmin:%f", data.Zmin, boundingBox.Zmin)
		}
		if boundingBox.Zmid != data.Zmid {
			t.Errorf("Expected Zmid:%f, got Zmid:%f", data.Zmid, boundingBox.Zmid)
		}
		if boundingBox.Zmax != data.Zmax {
			t.Errorf("Expected Zmax:%f, got Zmax:%f", data.Zmax, boundingBox.Zmax)
		}
	}
}

func TestGetWGS84Volume(t *testing.T) {
	testData := []struct {
		Xmin   float64
		Xmax   float64
		Ymin   float64
		Ymax   float64
		Zmin   float64
		Zmax   float64
		Volume float64
	}{
		{43.55994274887245, 43.56017598918163, 11.853911461356583, 11.85428160620063, 4.0, 5.0, 775.22},
		{43.55994274887245, 43.56017598918163, 11.853911461356583, 11.85428160620063, -5.0, 5.0, 7752.2},
		{43.55994274887245, 43.56017598918163, 11.853911461356583, 11.85428160620063, 5.0, 5.0, 0.0},
	}

	for _, data := range testData {
		boundingBox := geometry.NewBoundingBox(data.Xmin, data.Xmax, data.Ymin, data.Ymax, data.Zmin, data.Zmax)
		volume := boundingBox.GetWGS84Volume()
		if math.Abs(volume-data.Volume) > 1E-1 {
			t.Errorf("Expected Volume:%f, got Volume:%f", data.Volume, volume)
		}
	}
}
