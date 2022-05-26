// Copyright 2019 Massimo Federico Bonfigli

// This file contains definitions of helper functions to tailor the lidario library
// to the needs of the cesium tiler library

package lidario

import (
	"encoding/binary"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
)

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
		// recLengths := [4][4]int{{20, 18, 19, 17}, {28, 26, 27, 25}, {26, 24, 25, 23}, {34, 32, 33, 31}}

		// if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][0] {
		// 	las.usePointIntensity = true
		// 	las.usePointUserdata = true
		// } else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][1] {
		// 	las.usePointIntensity = false
		// 	las.usePointUserdata = true
		// } else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][2] {
		// 	las.usePointIntensity = true
		// 	las.usePointUserdata = false
		// } else if las.Header.PointRecordLength == recLengths[las.Header.PointFormatID][3] {
		// 	las.usePointIntensity = false
		// 	las.usePointUserdata = false
		// }

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
	pointsLength := las.Header.NumberPoints*las.Header.PointRecordLength + 5
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
				// p := PointRecord0{}
				X := float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.XScaleFactor + las.Header.XOffset
				offset += 4
				Y := float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.YScaleFactor + las.Header.YOffset
				offset += 4
				Z := float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.ZScaleFactor + las.Header.ZOffset
				offset += 4

				var R, G, B, Intensity, Classification uint8
				Intensity = uint8(binary.LittleEndian.Uint16(b[offset:offset+2]) / 256)
				offset += 2
				//p.BitField = PointBitField{Value: b[offset]}
				offset++
				if las.Header.PointFormatID >= 6 {
					offset++
				}
				//p.ClassBitField = ClassificationBitField{Value: b[offset]}
				Classification = b[offset]
				offset++
				// p.ScanAngle = int8(b[offset])
				offset++
				// point user data flag:
				offset++
				// p.PointSourceID = binary.LittleEndian.Uint16(b[offset : offset+2])
				offset += 2

				// las.pointData[i] = p

				if las.Header.PointFormatID == 1 || las.Header.PointFormatID >= 3 {
					// las.gpsData[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
					offset += 8
				}
				if las.Header.PointFormatID == 2 || las.Header.PointFormatID == 3 || las.Header.PointFormatID == 5 || las.Header.PointFormatID == 8 || las.Header.PointFormatID == 10 {
					var conversionFactor = uint16(256)
					if eightBitColor {
						conversionFactor = uint16(1)
					}
					//rgb := RgbData{}
					R = uint8(binary.LittleEndian.Uint16(b[offset:offset+2]) / conversionFactor)
					offset += 2
					G = uint8(binary.LittleEndian.Uint16(b[offset:offset+2]) / conversionFactor)
					offset += 2
					B = uint8(binary.LittleEndian.Uint16(b[offset:offset+2]) / conversionFactor)
					offset += 2
					// las.rgbData[i] = rgb
				}
				// TODO: Add support for other point formats with RGB color components
				lasFileLoader.Tree.AddPoint(&geometry.Coordinate{X: X, Y: Y, Z: Z}, R, G, B, Intensity, Classification, inSrid)
				// las.pointDataOctElement[i] = elem
			}
		}(startingPoint, endingPoint)
		startingPoint = endingPoint + 1
	}
	wg.Wait()
	return nil
}
