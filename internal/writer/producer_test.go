package writer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
)

func TestProduce(t *testing.T) {
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

	p := NewStandardProducer("path", "folder")
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
	c := make(chan *WorkUnit, 10)
	ec := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	p.Produce(c, ec, wg, root, context.TODO())
	wg.Wait()
	rootSeen := false
	childSeen := false
	for wu := range c {
		if wu.Node != root && wu.Node != child {
			t.Errorf("unexpected unit")
		}
		if wu.Node == root {
			rootSeen = true
			if wu.BasePath != "path/folder/data" {
				t.Errorf("unexpected path, expected path/folder/data, got %s", wu.BasePath)
			}
		}
		if wu.Node == child {
			childSeen = true
			if wu.BasePath != "path/folder/data" {
				t.Errorf("unexpected path, expected path/folder/data, got %s", wu.BasePath)
			}
		}
	}
	if !rootSeen || !childSeen {
		t.Errorf("not all nodes were seen")
	}
	if len(ec) != 0 {
		t.Errorf("unexpected errors in the channel")
	}
}

func TestProduceWithCancelOk(t *testing.T) {

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

	p := NewStandardProducer("path", "folder")
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
	c := make(chan *WorkUnit)
	ec := make(chan error, 10)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	mockErr := fmt.Errorf("mock error")
	ctx, _ := context.WithDeadlineCause(context.Background(), time.Now().Add(500*time.Millisecond), mockErr)
	time.Sleep(600 * time.Millisecond)
	p.Produce(c, ec, wg, root, ctx)
	wg.Wait()
	if len(c) > 0 {
		t.Errorf("unexpected work units in the channel")
	}
	if len(ec) == 0 {
		t.Errorf("expected errors in the channel")
	}
}
