// Copyright 2019 Massimo Federico Bonfigli

// This file contains definitions of helper functions to tailor the lidario library
// to the needs of the cesium tiler library

package lidario

import (
	"encoding/binary"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"io"
	"os"
	"runtime"
	"sync"
)

var recLengths = [11][4]int{
	{20, 18, 19, 17}, // Point format 0
	{28, 26, 27, 25}, // Point format 1
	{26, 24, 25, 23}, // Point format 2
	{34, 32, 33, 31}, // Point format 3
	{57, 55, 56, 54}, // Point format 4
	{63, 61, 62, 60}, // Point format 5
	{30, 28, 29, 27}, // Point format 6
	{36, 34, 35, 33}, // Point format 7
	{38, 36, 37, 35}, // Point format 8
	{59, 57, 58, 56}, // Point format 9
	{67, 65, 66, 64}, // Point format 10
}

var xyzOffets = [11][3]int{
	{0, 4, 8}, // Point format 0
	{0, 4, 8}, // Point format 1
	{0, 4, 8}, // Point format 2
	{0, 4, 8}, // Point format 3
	{0, 4, 8}, // Point format 4
	{0, 4, 8}, // Point format 5
	{0, 4, 8}, // Point format 6
	{0, 4, 8}, // Point format 7
	{0, 4, 8}, // Point format 8
	{0, 4, 8}, // Point format 9
	{0, 4, 8}, // Point format 10
}

var rgbOffets = [11][]int{
	nil, 			// Point format 0
	nil, 			// Point format 1
	{20, 22, 24}, 	// Point format 2
	{28, 30, 32}, 	// Point format 3
	nil, 			// Point format 4
	{28, 30, 32}, 	// Point format 5
	nil, 			// Point format 6
	{30, 32, 34}, 	// Point format 7
	{30, 32, 34}, 	// Point format 8
	nil, 			// Point format 9
	{30, 32, 34}, 	// Point format 10
}

// intensity offset is always 12

var classificationOffets = [11]int{
	15, // Point format 0
	15, // Point format 1
	15,	// Point format 2
	15, // Point format 3
	15, // Point format 4
	15, // Point format 5
	16, // Point format 6
	16, // Point format 7
	16, // Point format 8
	16,	// Point format 9
	16, // Point format 10
}

type LasFileLoader struct {
	Tree octree.ITree
}

func NewLasFileLoader(tree octree.ITree) *LasFileLoader {
	return &LasFileLoader{
		Tree: tree,
	}
}

// NewLasFile creates a new LasFile structure which stores the points data directly into Point instances
// which can be retrieved by index using the GetPoint function
func (lasFileLoader *LasFileLoader) LoadLasFile(fileName string, inSrid int, eightBitColor bool) (*LasFile, error) {
	// initialize the VLR array
	vlrs := []VLR{}
	las := LasFile{fileName: fileName, fileMode: "r", Header: LasHeader{}, VlrData: vlrs}
	if err := lasFileLoader.readForOctree(inSrid, eightBitColor, &las); err != nil {
		return &las, err
	}
	return &las, nil
}

// Reads the las file and produces a LasFile struct instance loading points data into its inner list of Point
func (lasFileLoader *LasFileLoader) readForOctree(inSrid int, eightBitColor bool, las *LasFile) error {
	var err error
	if las.f, err = os.Open(las.fileName); err != nil {
		return err
	}
	if err = las.readHeader(); err != nil {
		return err
	}
	if err := las.readVLRs(); err != nil {
		return err
	}
	if las.fileMode != "rh" {


		if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][0] {
			las.usePointIntensity = true
			las.usePointUserdata = true
		} else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][1] {
			las.usePointIntensity = false
			las.usePointUserdata = true
		} else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][2] {
			las.usePointIntensity = true
			las.usePointUserdata = false
		} else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][3] {
			las.usePointIntensity = false
			las.usePointUserdata = false
		}

		if err := lasFileLoader.readPointsOctElem(inSrid, eightBitColor, las); err != nil {
			return err
		}
	}
	return nil
}

// Reads all the points of the given las file and parses them into a Point data structure which is then stored
// in the given LasFile instance
func (lasFileLoader *LasFileLoader) readPointsOctElem(inSrid int, eightBitColor bool, las *LasFile) error {
	las.Lock()
	defer las.Unlock()
	// las.pointDataOctElement = make([]octree.OctElement, las.Header.NumberPoints)
	if las.Header.PointFormatID == 1 || las.Header.PointFormatID == 3 {
		// las.gpsData = make([]float64, las.Header.NumberPoints)
	}
	if las.Header.PointFormatID == 2 || las.Header.PointFormatID == 3 {
		// las.rgbData = make([]RgbData, las.Header.NumberPoints)
	}

	// Estimate how many bytes are used to store the points
	pointsLength := las.Header.NumberPoints * las.Header.PointRecordLength
	b := make([]byte, pointsLength)
	if _, err := las.f.ReadAt(b, int64(las.Header.OffsetToPoints)); err != nil && err != io.EOF {
		// return err
	}

    // The LAS Specifications state that:
	// " Point data items that are not ‘Required’ must be set to
	// the equivalent of zero for the data type (e.g. 0.0 for floating types, null for ASCII, 0 for integers)."
	//
	// In this context this means that basically the intensity/user data field is always present just with zero value.
	// As such the corresponding bytes are always considered when parsing the payload
	// The entire logic will probably need to be rewritten from scratch based on the
	// las file format specifications rather than bugfixing the original las read library logic
	// imported and used in this project.

	numCPUs := runtime.NumCPU()
	var wg sync.WaitGroup
	blockSize := las.Header.NumberPoints / numCPUs
	var startingPoint int
	for startingPoint < las.Header.NumberPoints {
		endingPoint := startingPoint + blockSize
		if endingPoint >= las.Header.NumberPoints {
			endingPoint = las.Header.NumberPoints - 1
		}
		wg.Add(1)
		go func(pointSt, pointEnd int) {
			defer wg.Done()

			var offset int
			// var p PointRecord0
			for i := pointSt; i <= pointEnd; i++ {
				offset = i * las.Header.PointRecordLength
				X, Y, Z, R, G, B, Intensity, Classification := readPoint(&las.Header, b, offset, eightBitColor)
				lasFileLoader.Tree.AddPoint(&geometry.Coordinate{X: X, Y: Y, Z: Z}, R, G, B, Intensity, Classification, inSrid)
				// las.pointDataOctElement[i] = elem
			}
		}(startingPoint, endingPoint)
		startingPoint = endingPoint + 1
	}
	wg.Wait()
	return nil
}


func readPoint(header *LasHeader, data []byte, offset int, eightBitColor bool) (float64, float64, float64, uint8, uint8, uint8, uint8, uint8) {
	var x, y, z float64
	var r, g, b uint8
	var intensity uint8
	var classification uint8
	xyzOffsetValues := xyzOffets[header.PointFormatID]
	xOffset := xyzOffsetValues[0] + offset
	yOffset := xyzOffsetValues[1] + offset
	zOffset := xyzOffsetValues[2] + offset
	x = float64(int32(binary.LittleEndian.Uint32(data[xOffset:xOffset+4])))*header.XScaleFactor + header.XOffset
	y = float64(int32(binary.LittleEndian.Uint32(data[yOffset:yOffset+4])))*header.YScaleFactor + header.YOffset
	z = float64(int32(binary.LittleEndian.Uint32(data[zOffset:zOffset+4])))*header.ZScaleFactor + header.ZOffset

	rgbOffsetValues := rgbOffets[header.PointFormatID]
	if rgbOffsetValues != nil {
		rOffset := rgbOffsetValues[0] + offset
		gOffset := rgbOffsetValues[1] + offset
		bOffset := rgbOffsetValues[2] + offset
		var conversionFactor = uint16(256)
		if eightBitColor {
			conversionFactor = uint16(1)
		}

		r = uint8(binary.LittleEndian.Uint16(data[rOffset:rOffset+2]) / conversionFactor)
		g = uint8(binary.LittleEndian.Uint16(data[gOffset:gOffset+2]) / conversionFactor)
		b = uint8(binary.LittleEndian.Uint16(data[bOffset:bOffset+2]) / conversionFactor)
	}
	intensityOffset := 12 + offset
	intensity = uint8(binary.LittleEndian.Uint16(data[intensityOffset:intensityOffset+2]) / 256)
	classificationOffset := classificationOffets[header.PointFormatID] + offset
	classification = data[classificationOffset]

	return x,y,z,r,g,b,intensity,classification
}