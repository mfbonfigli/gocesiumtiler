package writer

import (
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
)

func TestConsume(t *testing.T) {
	c := NewStandardConsumer()
	wc := make(chan *WorkUnit)
	ec := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go c.Consume(wc, ec, wg)

	pts := []model.Point{
		{X: 0, Y: 0, Z: 0, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
		{X: 1, Y: 3, Z: 4, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
		{X: 2, Y: 6, Z: 8, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
	}

	pt1 := &geom.LinkedPoint{
		Pt: pts[0],
	}
	pt2 := &geom.LinkedPoint{
		Pt: pts[1],
	}
	pt3 := &geom.LinkedPoint{
		Pt: pts[2],
	}
	pt1.Next = pt2
	pt2.Next = pt3

	stream := geom.NewLinkedPointStream(pt1, 3)
	tr := geom.LocalToGlobalTransformFromPoint(1000, 1000, 1000)
	n := &tree.MockNode{
		TotalNumPts: 3,
		Pts:         stream,
		Bounds: geom.NewBoundingBox(
			0,
			4,
			0,
			6,
			0,
			8,
		),
		Root:      true,
		Leaf:      true,
		GeomError: 20,
		Transform: &tr,
	}

	tmp, err := os.MkdirTemp(os.TempDir(), "tst")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	tmpPath := filepath.Join(tmp, "tst")
	os.Mkdir(tmpPath, os.ModeAppend)
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	wc <- &WorkUnit{
		Node:     n,
		BasePath: tmpPath,
	}
	close(wc)
	wg.Wait()

	actualPnts, err := os.ReadFile(filepath.Join(tmpPath, "d.pnts"))
	if err != nil {
		t.Fatalf("unable to read d.pnts: %v", err)
	}
	expectedPnts, err := os.ReadFile("./testdata/content.pnts")
	if err != nil {
		t.Fatalf("unable to read testdata/content.pnts: %v", err)
	}
	if !reflect.DeepEqual(actualPnts, expectedPnts) {
		t.Errorf("expected pnts:\n%v\n\ngot:\n\n%v\n", expectedPnts, actualPnts)
	}
}

func TestConsumeGltf(t *testing.T) {
	c := NewStandardConsumer(WithGeometryEncoder(NewGltfEncoder("d.glb")))
	wc := make(chan *WorkUnit)
	ec := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go c.Consume(wc, ec, wg)

	pts := []model.Point{
		{X: 0, Y: 0, Z: 0, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
		{X: 1, Y: 1, Z: 1, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
		{X: 2, Y: 2, Z: 2, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
	}

	pt1 := &geom.LinkedPoint{
		Pt: pts[0],
	}
	pt2 := &geom.LinkedPoint{
		Pt: pts[1],
	}
	pt3 := &geom.LinkedPoint{
		Pt: pts[2],
	}
	pt1.Next = pt2
	pt2.Next = pt3

	tr := geom.LocalToGlobalTransformFromPoint(2000, 1000, 1000)
	stream := geom.NewLinkedPointStream(pt1, 3)
	n := &tree.MockNode{
		TotalNumPts: 3,
		Pts:         stream,
		Bounds: geom.NewBoundingBox(
			0,
			4,
			0,
			6,
			0,
			8,
		),
		Root:      true,
		Leaf:      true,
		GeomError: 20,
		Transform: &tr,
	}

	tmp, err := os.MkdirTemp(os.TempDir(), "tst")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	tmpPath := filepath.Join(tmp, "tst")
	os.Mkdir(tmpPath, os.ModeAppend)
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	wc <- &WorkUnit{
		Node:     n,
		BasePath: tmpPath,
	}
	close(wc)
	wg.Wait()
	actualGlb, err := os.ReadFile(filepath.Join(tmpPath, "d.glb"))
	if err != nil {
		t.Fatalf("unable to read d.glb: %v", err)
	}
	expectedGlb, err := os.ReadFile("./testdata/content.glb")
	if err != nil {
		t.Fatalf("unable to read testdata/content.glb: %v", err)
	}
	if !reflect.DeepEqual(actualGlb, expectedGlb) {
		t.Errorf("expected glb:\n%v\n\ngot:\n\n%v\n", expectedGlb, actualGlb)
	}
}
