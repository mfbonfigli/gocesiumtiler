package writer

import (
	"context"
	"fmt"
	"testing"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
)

func TestWriter(t *testing.T) {
	pt1 := &geom.LinkedPoint{
		Pt: geom.NewPoint(1, 2, 3, 4, 5, 6, 7, 8),
	}
	pt2 := &geom.LinkedPoint{
		Pt: geom.NewPoint(9, 10, 11, 12, 13, 14, 15, 16),
	}
	pt3 := &geom.LinkedPoint{
		Pt: geom.NewPoint(17, 18, 19, 20, 21, 22, 23, 24),
	}
	pt1.Next = pt2
	pt2.Next = pt3

	stream := geom.NewLinkedPointStream(pt1, 3)
	stream2 := geom.NewLinkedPointStream(pt2, 2)

	child := &tree.MockNode{
		TotalNumPts: 2,
		Pts:         stream2,
	}
	root := &tree.MockNode{
		TotalNumPts: 5,
		Pts:         stream,
		ChildNodes: [8]tree.Node{
			nil,
			child,
		},
	}

	w, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	p := &MockProducer{}
	c := &MockConsumer{}
	w.producerFunc = func(basepath, folder string) Producer {
		return p
	}
	w.consumerFunc = func(v version.TilesetVersion) Consumer {
		return c
	}
	err = w.Write(root, "base", context.TODO())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if p.Wc == nil {
		t.Errorf("empty work channel passed")
	} else {
		if c.Wc != p.Wc {
			t.Errorf("passed different work channel to consumer")
		}
	}
	if p.Ec == nil {
		t.Errorf("empty error channel passed")
	} else {
		if c.Ec != p.Ec {
			t.Errorf("passed different error channel to consumer")
		}
	}
}

func TestWriterSquashTilesetContent(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create test data structure
	pt1 := &geom.LinkedPoint{
		Pt: geom.NewPoint(1, 2, 3, 4, 5, 6, 7, 8),
	}
	pt2 := &geom.LinkedPoint{
		Pt: geom.NewPoint(9, 10, 11, 12, 13, 14, 15, 16),
	}
	pt1.Next = pt2

	stream := geom.NewLinkedPointStream(pt1, 2)
	stream2 := geom.NewLinkedPointStream(pt2, 1)

	// Create a mock tree with child nodes
	child := &tree.MockNode{
		TotalNumPts: 1,
		Pts:         stream2,
		Bounds:      geom.NewBoundingBox(5, 6, 7, 8, 9, 10),
		GeomError:   10.0,
	}
	root := &tree.MockNode{
		TotalNumPts: 2,
		Pts:         stream,
		Bounds:      geom.NewBoundingBox(1, 2, 3, 4, 5, 6),
		GeomError:   20.0,
		Root:        true,
		ChildNodes: [8]tree.Node{
			nil,
			child,
		},
	}

	// Test squash mode - generates single tileset.json
	w, err := NewWriter(tmpDir,
		WithNumWorkers(1),
		WithBufferRatio(10),
		WithSquash(true),
		WithTilesetVersion(version.TilesetVersion_1_0),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// Test that tileset is generated correctly
	tileset := w.buildTileTree(root, "")

	// Verify root tile properties
	if tileset.GeometricError != 20.0 {
		t.Errorf("expected geometric error 20.0, got %v", tileset.GeometricError)
	}

	if tileset.Refine != "ADD" {
		t.Errorf("expected refine to be ADD, got %v", tileset.Refine)
	}

	if tileset.Content == nil {
		t.Errorf("expected root tile to have content")
	} else if tileset.Content.Url != "content.pnts" {
		t.Errorf("expected content URL to be 'content.pnts', got %v", tileset.Content.Url)
	}

	// Verify bounding box
	expectedBox := []float64{1, 2, 3, 4, 5, 6, 0, 0, 0, 0, 0, 0}
	if len(tileset.BoundingVolume.Box) != len(expectedBox) {
		t.Errorf("expected bounding box length %d, got %d", len(expectedBox), len(tileset.BoundingVolume.Box))
	}

	// Verify children exist
	if len(tileset.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(tileset.Children))
	}

	// Verify child properties
	if len(tileset.Children) > 0 {
		childTile := tileset.Children[0]
		if childTile.GeometricError != 10.0 {
			t.Errorf("expected child geometric error 10.0, got %v", childTile.GeometricError)
		}

		if childTile.Content == nil {
			t.Errorf("expected child tile to have content")
		} else if childTile.Content.Url != "1/content.pnts" {
			t.Errorf("expected child content URL to be '1/content.pnts', got %v", childTile.Content.Url)
		}
	}
}

func TestWriterWithProducerError(t *testing.T) {
	pt1 := &geom.LinkedPoint{
		Pt: geom.NewPoint(1, 2, 3, 4, 5, 6, 7, 8),
	}
	pt2 := &geom.LinkedPoint{
		Pt: geom.NewPoint(9, 10, 11, 12, 13, 14, 15, 16),
	}
	pt3 := &geom.LinkedPoint{
		Pt: geom.NewPoint(17, 18, 19, 20, 21, 22, 23, 24),
	}
	pt1.Next = pt2
	pt2.Next = pt3

	stream := geom.NewLinkedPointStream(pt1, 3)
	stream2 := geom.NewLinkedPointStream(pt2, 2)

	child := &tree.MockNode{
		TotalNumPts: 2,
		Pts:         stream2,
	}
	root := &tree.MockNode{
		TotalNumPts: 5,
		Pts:         stream,
		ChildNodes: [8]tree.Node{
			nil,
			child,
		},
	}

	w, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	p := &MockProducer{
		Err: fmt.Errorf("mock error"),
	}
	c := &MockConsumer{}
	w.producerFunc = func(basepath, folder string) Producer {
		return p
	}
	w.consumerFunc = func(v version.TilesetVersion) Consumer {
		return c
	}
	err = w.Write(root, "base", context.TODO())
	if err == nil {
		t.Errorf("expected error but got none")
	}
	if p.Wc == nil {
		t.Errorf("empty work channel passed")
	} else {
		if c.Wc != p.Wc {
			t.Errorf("passed different work channel to consumer")
		}
	}
	if p.Ec == nil {
		t.Errorf("empty error channel passed")
	} else {
		if c.Ec != p.Ec {
			t.Errorf("passed different error channel to consumer")
		}
	}
}

func TestWriterWithConsumerError(t *testing.T) {
	pt1 := &geom.LinkedPoint{
		Pt: geom.NewPoint(1, 2, 3, 4, 5, 6, 7, 8),
	}
	pt2 := &geom.LinkedPoint{
		Pt: geom.NewPoint(9, 10, 11, 12, 13, 14, 15, 16),
	}
	pt3 := &geom.LinkedPoint{
		Pt: geom.NewPoint(17, 18, 19, 20, 21, 22, 23, 24),
	}
	pt1.Next = pt2
	pt2.Next = pt3

	stream := geom.NewLinkedPointStream(pt1, 3)
	stream2 := geom.NewLinkedPointStream(pt2, 2)

	child := &tree.MockNode{
		TotalNumPts: 2,
		Pts:         stream2,
	}
	root := &tree.MockNode{
		TotalNumPts: 5,
		Pts:         stream,
		ChildNodes: [8]tree.Node{
			nil,
			child,
		},
	}

	w, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	p := &MockProducer{}
	c := &MockConsumer{
		Err: fmt.Errorf("mock error"),
	}
	w.producerFunc = func(basepath, folder string) Producer {
		return p
	}
	w.consumerFunc = func(v version.TilesetVersion) Consumer {
		return c
	}
	err = w.Write(root, "base", context.TODO())
	if err == nil {
		t.Errorf("expected error but got none")
	}
	if p.Wc == nil {
		t.Errorf("empty work channel passed")
	} else {
		if c.Wc != p.Wc {
			t.Errorf("passed different work channel to consumer")
		}
	}
	if p.Ec == nil {
		t.Errorf("empty error channel passed")
	} else {
		if c.Ec != p.Ec {
			t.Errorf("passed different error channel to consumer")
		}
	}
}

func TestWriterTilesetVersion(t *testing.T) {
	w, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
		WithTilesetVersion(version.TilesetVersion_1_0),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if w.version != version.TilesetVersion_1_0 {
		t.Errorf("unexpected tileset version")
	}
	c := w.consumerFunc(version.TilesetVersion_1_0)
	if _, success := (c.(*StandardConsumer).encoder).(*PntsEncoder); success != true {
		t.Errorf("unexpected geometry encoder for tileset version 1.0")
	}
	w, err = NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
		WithTilesetVersion(version.TilesetVersion_1_1),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if w.version != version.TilesetVersion_1_1 {
		t.Errorf("unexpected tileset version")
	}
	c = w.consumerFunc(version.TilesetVersion_1_1)
	if _, success := (c.(*StandardConsumer).encoder).(*GltfEncoder); success != true {
		t.Errorf("unexpected geometry encoder for tileset version 1.1")
	}
}

func TestWriterSquashMode(t *testing.T) {
	// Create test data structure
	pt1 := &geom.LinkedPoint{
		Pt: geom.NewPoint(1, 2, 3, 4, 5, 6, 7, 8),
	}
	pt2 := &geom.LinkedPoint{
		Pt: geom.NewPoint(9, 10, 11, 12, 13, 14, 15, 16),
	}
	pt3 := &geom.LinkedPoint{
		Pt: geom.NewPoint(17, 18, 19, 20, 21, 22, 23, 24),
	}
	pt1.Next = pt2
	pt2.Next = pt3

	stream := geom.NewLinkedPointStream(pt1, 3)
	stream2 := geom.NewLinkedPointStream(pt2, 2)

	// Create a mock tree with child nodes
	child := &tree.MockNode{
		TotalNumPts: 2,
		Pts:         stream2,
		Bounds:      geom.NewBoundingBox(5, 6, 7, 8, 9, 10),
		GeomError:   10.0,
	}
	_ = &tree.MockNode{
		TotalNumPts: 5,
		Pts:         stream,
		Bounds:      geom.NewBoundingBox(1, 2, 3, 4, 5, 6),
		GeomError:   20.0,
		ChildNodes: [8]tree.Node{
			nil,
			child,
		},
	}

	// Test squash mode enabled
	w, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
		WithSquash(true),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// Verify squash flag is set
	if w.squash != true {
		t.Errorf("expected squash to be true, got %v", w.squash)
	}

	// Verify consumer has skipTileset set to true
	c := w.consumerFunc(version.TilesetVersion_1_0)
	if stdConsumer, ok := c.(*StandardConsumer); ok {
		if stdConsumer.skipTileset != true {
			t.Errorf("expected consumer skipTileset to be true when squash is enabled")
		}
	} else {
		t.Errorf("expected StandardConsumer type")
	}

	// Test squash mode disabled (default)
	w2, err := NewWriter("base",
		WithNumWorkers(1),
		WithBufferRatio(10),
		WithSquash(false),
	)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// Verify squash flag is not set
	if w2.squash != false {
		t.Errorf("expected squash to be false, got %v", w2.squash)
	}

	// Verify consumer has skipTileset set to false
	c2 := w2.consumerFunc(version.TilesetVersion_1_0)
	if stdConsumer, ok := c2.(*StandardConsumer); ok {
		if stdConsumer.skipTileset != false {
			t.Errorf("expected consumer skipTileset to be false when squash is disabled")
		}
	} else {
		t.Errorf("expected StandardConsumer type")
	}
}
