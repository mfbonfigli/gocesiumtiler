package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/proj4_coordinate_converter"
	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/io"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"path"
	"sync"
	"testing"
)

func TestProducerInjectsWorkUnits(t *testing.T) {
	var opts = tiler.TilerOptions{
		Srid:                4326,
		CoordinateConverter: proj4_coordinate_converter.NewProj4CoordinateConverter(),
	}

	rootNode := &mockNode{
		boundingBox: geometry.NewBoundingBox(13.7995147, 13.7995147, 42.3306312, 42.3306312, 0, 1),
		points: []*data.Point{
			data.NewPoint(13.7995147, 42.3306312, 1, 1, 2, 3, 4, 5),
		},
		depth:               1,
		globalChildrenCount: 2,
		localChildrenCount:  1,
		initialized:         true,
		opts:                &opts,
		children: [8]octree.INode{
			&mockNode{
				boundingBox: geometry.NewBoundingBox(13.7995147, 13.7995147, 42.3306312, 42.3306312, 0.5, 1),
				points: []*data.Point{
					data.NewPoint(13.7995147, 42.3306312, 1, 4, 5, 6, 4, 5),
				},
				depth:               1,
				globalChildrenCount: 1,
				localChildrenCount:  1,
				initialized:         true,
				opts:                &opts,
			},
		},
	}

	workChannel := make(chan *io.WorkUnit, 3)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	io.Produce("basepath", rootNode, &opts, workChannel, &waitGroup, "")
	waitGroup.Wait() // if the test waits here indefinitely then producer is not deregistering itself from the waitgroup with waitGroup.Done()

	if len(workChannel) != 2 {
		t.Errorf("Expected to find %d items in the workchannel but %d were found", 2, len(workChannel))
	}

	rootWorkUnit := <-workChannel
	if rootWorkUnit.OctNode != rootNode {
		t.Errorf("Missing root node in workchannel")
	}
	if rootWorkUnit.BasePath != "basepath" {
		t.Errorf("Expected basepath: %s got %s", "basepath", rootWorkUnit.BasePath)
	}
	if rootWorkUnit.Opts != &opts {
		t.Errorf("Missing expected tiler options")
	}

	childWorkUnit := <-workChannel
	if childWorkUnit.OctNode != rootNode.children[0] {
		t.Errorf("Missing child node in workchannel")
	}
	if childWorkUnit.BasePath != path.Join("basepath", "0") {
		t.Errorf("Expected basepath: %s got %s", path.Join("basepath", "0"), childWorkUnit.BasePath)
	}
	if childWorkUnit.Opts != &opts {
		t.Errorf("Missing expected tiler options")
	}

	select {
	case <-workChannel:
	default:
		t.Error("Producer didn't close the WorkChannel was not closed")
	}

}
