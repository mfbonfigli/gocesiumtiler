package writer

import (
	"encoding/json"
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
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	tmpPath := filepath.Join(tmp, "tst")

	wc <- &WorkUnit{
		Node:     n,
		BasePath: tmpPath,
	}
	close(wc)
	wg.Wait()
	sb, err := os.ReadFile(filepath.Join(tmpPath, "tileset.json"))
	if err != nil {
		t.Fatalf("unable to read tileset.json: %v", err)
	}
	expTrans := tr.ForwardColumnMajor()
	expected := Tileset{
		Asset: Asset{
			Version: "1.0",
		},
		GeometricError: 20,
		Root: Root{
			Children: nil,
			Content: &Content{
				Url: "content.pnts",
			},
			BoundingVolume: BoundingVolume{
				Box: [12]float64{
					2,
					3,
					4,
					2, 0, 0,
					0, 3, 0,
					0, 0, 4,
				},
			},
			GeometricError: 20,
			Refine:         "ADD",
			Transform:      &expTrans,
		},
	}

	actual := Tileset{}
	err = json.Unmarshal(sb, &actual)
	if err != nil {
		t.Fatalf("unable to decode tileset.json: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("unexpected tileset.json, expected:\n*%v*\n\ngot:\n\n*%v*\n", expected, actual)
	}

	actualPnts, err := os.ReadFile(filepath.Join(tmpPath, "content.pnts"))
	if err != nil {
		t.Fatalf("unable to read content.pnts: %v", err)
	}
	expectedPnts, err := os.ReadFile("./testdata/content.pnts")
	if err != nil {
		t.Fatalf("unable to read tileset.json: %v", err)
	}
	if !reflect.DeepEqual(actualPnts, expectedPnts) {
		t.Errorf("expected pnts:\n%v\n\ngot:\n\n%v\n", expectedPnts, actualPnts)
	}
}

func TestConsumeGltf(t *testing.T) {
	c := NewStandardConsumer(WithGeometryEncoder(NewGltfEncoder()))
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
	expTrans := tr.ForwardColumnMajor()
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
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	tmpPath := filepath.Join(tmp, "tst")

	wc <- &WorkUnit{
		Node:     n,
		BasePath: tmpPath,
	}
	close(wc)
	wg.Wait()
	sb, err := os.ReadFile(filepath.Join(tmpPath, "tileset.json"))
	if err != nil {
		t.Fatalf("unable to read tileset.json: %v", err)
	}
	expected := Tileset{
		Asset: Asset{
			Version: "1.1",
		},
		GeometricError: 20,
		Root: Root{
			Children: nil,
			Content: &Content{
				Url: "content.glb",
			},
			BoundingVolume: BoundingVolume{
				Box: [12]float64{
					2,
					3,
					4,
					2, 0, 0,
					0, 3, 0,
					0, 0, 4,
				},
			},
			GeometricError: 20,
			Refine:         "ADD",
			Transform:      &expTrans,
		},
	}

	actual := Tileset{}
	err = json.Unmarshal(sb, &actual)
	if err != nil {
		t.Fatalf("unable to decode tileset.json: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("unexpected tileset.json, expected:\n*%v*\n\ngot:\n\n*%v*\n", expected, actual)
	}

	actualGlb, err := os.ReadFile(filepath.Join(tmpPath, "content.glb"))
	if err != nil {
		t.Fatalf("unable to read content.pnts: %v", err)
	}
	expectedGlb, err := os.ReadFile("./testdata/content.glb")
	if err != nil {
		t.Fatalf("unable to read tileset.json: %v", err)
	}
	if !reflect.DeepEqual(actualGlb, expectedGlb) {
		t.Errorf("expected glb:\n%v\n\ngot:\n\n%v\n", expectedGlb, actualGlb)
	}
}

func TestGenerateTilesetChild(t *testing.T) {
	c := NewStandardConsumer(WithGeometryEncoder(NewGltfEncoder())).(*StandardConsumer)
	node := tree.MockNode{
		Bounds:    geom.NewBoundingBox(0, 10, 0, 10, 0, 10),
		GeomError: 2.5,
	}
	out := c.generateTilesetChild(&node, 2)
	expectedBox := [12]float64{5, 5, 5, 5, 0, 0, 0, 5, 0, 0, 0, 5}
	if actual := out.BoundingVolume.Box; actual != expectedBox {
		t.Errorf("expected box %v, got %v", expectedBox, actual)
	}
	expectedContentUrl := "2/tileset.json"
	if actual := out.Content.Url; actual != expectedContentUrl {
		t.Errorf("expected url %v, got %v", expectedContentUrl, actual)
	}
	expectedGeomError := 2.5
	if actual := out.GeometricError; actual != expectedGeomError {
		t.Errorf("expected geom err %v, got %v", expectedGeomError, actual)
	}
	if actual := out.Refine; actual != "ADD" {
		t.Errorf("expected refine mode ADD, got %v", actual)
	}
}
