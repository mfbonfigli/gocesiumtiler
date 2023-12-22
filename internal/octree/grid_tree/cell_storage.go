package grid_tree

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/data"
)

// cellStorage models a storage for cell point data
type cellStorage interface {
	// getPoints return all points stored in this cell
	getPoints() []*data.Point
	// storeAndPush stores the given point and pushes out the first of the points previously stored
	storeAndPush(*data.Point) *data.Point
	// store the given point
	store(*data.Point)
	// isEmpty returns true if no point is stored in this cell
	isEmpty() bool
}

// memoryBackedCellStorage is a cell storage that stores all points in memory
type memoryBackedCellStorage struct {
	points []*data.Point
	sync.RWMutex
}

func (m *memoryBackedCellStorage) getPoints() []*data.Point {
	m.RLock()
	defer m.RUnlock()
	return m.points
}

func (m *memoryBackedCellStorage) storeAndPush(pt *data.Point) *data.Point {
	m.Lock()
	defer m.Unlock()
	if len(m.points) == 0 {
		m.points = append(m.points, pt)
		return nil
	}
	old := m.points[0]
	m.points[0] = pt
	return old
}

func (m *memoryBackedCellStorage) store(pt *data.Point) {
	m.Lock()
	defer m.Unlock()
	m.points = append(m.points, pt)
}

func (m *memoryBackedCellStorage) isEmpty() bool {
	m.RLock()
	defer m.RUnlock()
	return len(m.points) == 0
}

// memoryBackedCellStorage is a cell storage that stores all points inside a temporary file
type diskBackedCellStorage struct {
	cellTempFileName string
	sync.RWMutex
}

func (d *diskBackedCellStorage) getPoints() []*data.Point {
	d.RLock()
	defer d.RUnlock()
	pts := []*data.Point{}
	open, err := os.Open(d.cellTempFileName)
	if err != nil {
		log.Fatalf("unable to open the temporary file to get points %s: %v", err)
	}
	defer open.Close()
	r := bufio.NewReader(open)
	line := ""
	err = nil
	for err == nil {
		line, err = r.ReadString('\n')
		if len(line) == 0 {
			continue
		}
		pt := &data.Point{}
		err := json.Unmarshal([]byte(line), pt)
		if err != nil {
			log.Fatalf("unable to unmarshal temporary data for %s with value %s: %v", d.cellTempFileName, line, err)
		}
		pts = append(pts, pt)
	}
	return pts
}

func (d *diskBackedCellStorage) storeAndPush(pt *data.Point) *data.Point {
	d.Lock()
	defer d.Unlock()
	marshaledPt, err := json.Marshal(pt)
	if err != nil {
		log.Fatalf("unable to marshal point %v: %v", pt, err)
	}
	if _, err := os.Stat(d.cellTempFileName); err != nil {
		tmp, err := os.OpenFile(d.cellTempFileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("unable to create file %s: %v", d.cellTempFileName, err)
		}
		tmp.Write(marshaledPt)
		tmp.Write([]byte{'\n'})
		tmp.Close()
		return nil
	}
	source, err := os.Open(d.cellTempFileName)
	if err != nil {
		log.Fatalf("unable to open the temporary file %s: %v", d.cellTempFileName, err)
	}
	tempFilename := fmt.Sprintf("%s-tmp", d.cellTempFileName)
	tmp, err := os.OpenFile(tempFilename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("unable to create temp file %s: %v", tempFilename, err)
	}
	r := bufio.NewReader(source)
	count := 0
	line := ""
	err = nil
	var pushedOutPt = &data.Point{}
	for err == nil {
		line, err = r.ReadString('\n')
		if count == 0 {
			tmp.Write(marshaledPt)
			tmp.Write([]byte{'\n'})
			e := json.Unmarshal([]byte(line), pushedOutPt)
			if e != nil {
				log.Fatalf("unable to unmarshal line %s: %v", line, err)
			}
		} else {
			tmp.Write([]byte(line))
		}
		count += 1
	}
	source.Close()
	tmp.Close()
	os.Remove(d.cellTempFileName)
	os.Rename(tempFilename, d.cellTempFileName)
	return pushedOutPt
}

func (d *diskBackedCellStorage) store(pt *data.Point) {
	d.Lock()
	defer d.Unlock()
	marshaledPt, err := json.Marshal(pt)
	if err != nil {
		log.Fatalf("unable to marshal point %v: %v", pt, err)
	}
	f, err := os.OpenFile(d.cellTempFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("unable to open the file %s: %v", d.cellTempFileName, err)
	}
	f.Write(marshaledPt)
	f.Write([]byte{'\n'})
	f.Close()
}

func (d *diskBackedCellStorage) isEmpty() bool {
	d.RLock()
	defer d.RUnlock()
	if _, err := os.Stat(d.cellTempFileName); err == nil {
		return false
	}
	return true
}
