package grid

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/conv/coor"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/mutator"
)

// loader is a utility class used to read points from a las reader and initialize
// a grid Node with the correct data
type loader struct {
	createCoorConverter coor.ConverterFactory
	mutator             mutator.Mutator
	workers             int
}

// load the points from the LasReader r into the node n and then closes the reader
func (l *loader) load(n *Node, r las.LasReader, ctx context.Context) (err error) {
	defer r.Close()
	numPts := r.NumberOfPoints()
	if numPts == 0 {
		return fmt.Errorf("las with no points")
	}
	subCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// Store all the points in a continuous memory space
	// While not required, storing points in a contiguous array makes
	// the system more CPU cache friendly and thus measurably faster
	backingArray := make([]geom.LinkedPoint, numPts)

	// read first point

	c, err := l.createCoorConverter()
	if err != nil {
		return err
	}
	defer c.Cleanup()

	// compute the baseline point and local CRS
	localToGlobal, base, read, err := l.baseline(r, c)
	if err != nil {
		return err
	}
	backingArray[0] = base

	// init concurrent vars
	var wg sync.WaitGroup
	var errchan chan error = make(chan error)

	// launch consumers
	basePtPerWorker := (numPts - read) / l.workers
	residual := (numPts - read) - basePtPerWorker*l.workers
	start := read
	consumers := []*consumer{}
	for i := 0; i < l.workers; i++ {
		workerPtsNum := basePtPerWorker
		if i < residual {
			workerPtsNum += 1
		}
		conv, err := l.createCoorConverter()
		if err != nil {
			cancelFunc()
			return err
		}
		defer conv.Cleanup()
		consumers = append(consumers, newConsumer(i, start, workerPtsNum, conv, l.mutator, r.GetCRS(), &backingArray))
		wg.Add(1)
		go consumers[i].consume(r, localToGlobal, errchan, &wg, subCtx)
		start = start + workerPtsNum
	}

	// retrieve errors
	errs := []error{}
	errWg := &sync.WaitGroup{}
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for {
			err, ok := <-errchan
			if !ok {
				return
			}
			// in case of errors, cancel the context to abort early
			errs = append(errs, err)
			cancelFunc()
		}
	}()
	wg.Wait()
	close(errchan)
	errWg.Wait()

	if len(errs) != 0 {
		return errs[0]
	}

	var pts *geom.LinkedPoint
	var bboxbuilder *boundingBoxBuilder

	// merge consumer points and bounding boxes
	for _, c := range consumers {
		c.endPt.Next = pts
		pts = c.startPt
		if bboxbuilder == nil {
			bboxbuilder = c.bboxBuilder
		} else {
			bboxbuilder.mergeWith(c.bboxBuilder)
		}
	}
	bboxbuilder.processPoint(base.Pt.X, base.Pt.Y, base.Pt.Z)
	bbox := bboxbuilder.build()
	base.Next = pts

	// set data into the gridnode
	n.bounds = bbox
	n.pts = &base
	n.config.localToGlobal = &localToGlobal
	return nil
}

// toLocal is a utility function that transforms a Point64 to a locally referenced model.Point using
// the givn local to global transformation object
func toLocal(p geom.Point64, localToGlobal model.Transform) model.Point {
	localCoords := localToGlobal.Inverse(p.Vector)
	return geom.NewPoint(
		float32(localCoords.X),
		float32(localCoords.Y),
		float32(localCoords.Z),
		p.R,
		p.G,
		p.B,
		p.Intensity,
		p.Classification,
	)
}

// baseline fetches the first non-discarded point (mutators can discard points) and returns:
// - The transform from EPSG 4978 a zenithal local CRS centered in the point
// - the point, in local coordinates
// - the number of points read from the point cloud
// - An error in case the operation failed
func (l *loader) baseline(r las.LasReader, c coor.Converter) (model.Transform, geom.LinkedPoint, int, error) {
	read := 0
	for {
		first, err := r.GetNext()
		if err != nil {
			return model.Transform{}, geom.LinkedPoint{}, 0, err
		}
		read++
		pt, err := transformPoint(first, c, r.GetCRS())
		if err != nil {
			return model.Transform{}, geom.LinkedPoint{}, 0, err
		}

		localToGlobal := geom.LocalToGlobalTransformFromPoint(pt.X, pt.Y, pt.Z)
		baselinePtLocalCoords := toLocal(pt, localToGlobal)
		keep := true
		if l.mutator != nil {
			baselinePtLocalCoords, keep = l.mutator.Mutate(baselinePtLocalCoords, localToGlobal)
			if !keep {
				continue
			}
		}
		return localToGlobal, geom.LinkedPoint{Pt: baselinePtLocalCoords}, read, nil
	}
}

// consumer consumes points from the las reader storing them internally and updating its internal bounds
type consumer struct {
	id           int
	start        int
	count        int
	conv         coor.Converter
	mutator      mutator.Mutator
	crs          string
	backingArray *[]geom.LinkedPoint
	// output vars
	bboxBuilder *boundingBoxBuilder
	startPt     *geom.LinkedPoint
	endPt       *geom.LinkedPoint
}

func newConsumer(id int, start int, count int, conv coor.Converter, mut mutator.Mutator, crs string, backingArray *[]geom.LinkedPoint) *consumer {
	return &consumer{
		id:           id,
		start:        start,
		count:        count,
		conv:         conv,
		mutator:      mut,
		crs:          crs,
		backingArray: backingArray,
		bboxBuilder:  newBoundingBoxBuilder(),
	}
}

func (c *consumer) consume(r las.LasReader, localToGlobal model.Transform, errchan chan error, wg *sync.WaitGroup, ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			errchan <- fmt.Errorf("panic while reading from las: %v", r)
		}
	}()
	defer c.conv.Cleanup()
	defer wg.Done()
	var currentPt *geom.LinkedPoint
	i := 0
	for i < c.count {
		if err := ctx.Err(); err != nil {
			errchan <- err
			return
		}
		pt, err := r.GetNext()
		if err != nil {
			errchan <- err
			return
		}

		pt, err = transformPoint(pt, c.conv, c.crs)
		if err != nil {
			errchan <- err
			return
		}

		// transform local and update bounds
		localPt := toLocal(pt, localToGlobal)
		keep := true

		// mutate the point
		if c.mutator != nil {
			localPt, keep = c.mutator.Mutate(localPt, localToGlobal)
			if !keep {
				// point should be discarded, move on
				i++
				continue
			}
		}
		c.bboxBuilder.processPoint(localPt.X, localPt.Y, localPt.Z)

		// store point in backing array at the right offset
		(*c.backingArray)[c.start+i] = geom.LinkedPoint{Pt: localPt}
		newPt := &((*c.backingArray)[c.start+i])
		if currentPt == nil {
			currentPt = newPt
			c.startPt = currentPt
		} else {
			currentPt.Next = newPt
			currentPt = newPt
			c.endPt = currentPt
		}
		i++
	}
}

// transformPoint converts a point from the original CRS to EPSG:4978
func transformPoint(pt geom.Point64, conv coor.Converter, sourceCRS string) (geom.Point64, error) {
	var err error

	out, err := conv.ToWGS84Cartesian(sourceCRS, model.Vector{X: pt.X, Y: pt.Y, Z: pt.Z})
	if err != nil {
		return pt, err
	}

	pt.X, pt.Y, pt.Z = out.X, out.Y, out.Z
	return pt, nil
}

// boundingBoxBuilder is a utility struct to compute bounds from input points
type boundingBoxBuilder struct {
	minX, minY, minZ, maxX, maxY, maxZ float64
}

func newBoundingBoxBuilder() *boundingBoxBuilder {
	return &boundingBoxBuilder{
		minX: math.Inf(1),
		minY: math.Inf(1),
		minZ: math.Inf(1),
		maxX: math.Inf(-1),
		maxY: math.Inf(-1),
		maxZ: math.Inf(-1),
	}
}

// processPoint examines the input coordinates and expands the bounds if necessary
func (b *boundingBoxBuilder) processPoint(x, y, z float32) {
	if float64(x) < b.minX {
		b.minX = float64(x)
	}
	if float64(y) < b.minY {
		b.minY = float64(y)
	}
	if float64(z) < b.minZ {
		b.minZ = float64(z)
	}
	if float64(x) > b.maxX {
		b.maxX = float64(x)
	}
	if float64(y) > b.maxY {
		b.maxY = float64(y)
	}
	if float64(z) > b.maxZ {
		b.maxZ = float64(z)
	}
}

// mergeWith merges the current boundingBoxBuilder bounds with the input
// boundingBoxBuilder
func (b *boundingBoxBuilder) mergeWith(o *boundingBoxBuilder) {
	if o.minX < b.minX {
		b.minX = o.minX
	}
	if o.minY < b.minY {
		b.minY = o.minY
	}
	if o.minZ < b.minZ {
		b.minZ = o.minZ
	}
	if o.maxX > b.maxX {
		b.maxX = o.maxX
	}
	if o.maxY > b.maxY {
		b.maxY = o.maxY
	}
	if o.maxZ > b.maxZ {
		b.maxZ = o.maxZ
	}
}

// build returns a BoundingBox instance from the current builder bounds
func (b *boundingBoxBuilder) build() geom.BoundingBox {
	return geom.NewBoundingBox(b.minX, b.maxX, b.minY, b.maxY, b.minZ, b.maxZ)
}
