package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/point_loader"
	"reflect"
	"testing"
)

func TestSequentialLoaderAddPoint(t *testing.T) {
	loader := point_loader.NewSequentialLoader()
	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)

	loader.AddPoint(point)

	returnedPoint, hasNext := loader.GetNext()

	if hasNext {
		t.Errorf("Expected no further points to return")
	}

	if returnedPoint == nil {
		t.Errorf("Unexpected nil returned point ")
	}

	if returnedPoint != point {
		t.Errorf("Returned point different from input one")
	}
}

func TestSequentialLoaderGetNext(t *testing.T) {
	loader := point_loader.NewSequentialLoader()
	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	point2 := data.NewPoint(15, 45, 3, 2, 3, 4, 5, 6)

	loader.AddPoint(point)
	loader.AddPoint(point2)

	returnedPoint, hasNext := loader.GetNext()

	if !hasNext {
		t.Errorf("Expected one more point to return")
	}

	if returnedPoint == nil {
		t.Errorf("Unexpected nil returned point ")
	}

	if returnedPoint != point {
		t.Errorf("Returned point different from input one")
	}

	returnedPoint, hasNext = loader.GetNext()

	if hasNext {
		t.Errorf("Expected no more point to return")
	}

	if returnedPoint == nil {
		t.Errorf("Unexpected nil returned point ")
	}

	if returnedPoint != point2 {
		t.Errorf("Returned point different from input one")
	}
}

func TestSequentialLoaderGetBounds(t *testing.T) {
	loader := point_loader.NewSequentialLoader()
	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	point2 := data.NewPoint(15, 45, 3, 2, 3, 4, 5, 6)
	point3 := data.NewPoint(14.3, 47, 2, 2, 3, 4, 5, 6)

	loader.AddPoint(point)
	loader.AddPoint(point2)
	loader.AddPoint(point3)

	bounds := loader.GetBounds()
	expected := []float64{14.1, 15, 41, 47, 1, 3}

	if !reflect.DeepEqual(bounds, expected) {
		t.Errorf("Wrong loader bounds returned")
	}
}
