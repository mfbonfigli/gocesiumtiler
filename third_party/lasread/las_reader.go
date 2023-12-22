// Copyright 2019 Massimo Federico Bonfigli

// This file contains a custom wrapper for LasFile to read las points in random order

package lidario

import (
	"encoding/binary"
	"log"
	"os"
)

type LasReader struct {
	las           *LasFile
	eightBitColor bool
}

type LasPoint struct {
	X     float64
	Y     float64
	Z     float64
	R     uint8
	G     uint8
	B     uint8
	Int   uint8
	Class uint8
}

func (l *LasReader) GetNumPoints() int {
	return l.las.Header.NumberPoints
}

func (l *LasReader) GetPointAt(index int) *LasPoint {
	return l.readPoint(
		l.las,
		l.las.Header.OffsetToPoints+index*l.las.Header.PointRecordLength,
	)
}

func NewLasReader(fileName string, inSrid int, eightBitColor bool) (*LasReader, error) {
	lasFile, err := loadLasFile(fileName, inSrid, eightBitColor)
	if err != nil {
		return nil, err
	}
	return &LasReader{
		las:           lasFile,
		eightBitColor: eightBitColor,
	}, nil
}

// LoadLasFile initializes the LasReader with the given file. Return an error in case the file could
// not be opened or the headers read.
func loadLasFile(fileName string, inSrid int, eightBitColor bool) (*LasFile, error) {
	// initialize the VLR array
	vlrs := []VLR{}
	las := LasFile{fileName: fileName, fileMode: "r", Header: LasHeader{}, VlrData: vlrs}
	var err error
	if las.f, err = os.Open(las.fileName); err != nil {
		return nil, err
	}
	if err = las.readHeader(); err != nil {
		return nil, err
	}
	if err := las.readVLRs(); err != nil {
		return nil, err
	}
	return &las, nil
}

func (l *LasReader) readPoint(las *LasFile, offset int) *LasPoint {
	header := las.Header
	data := make([]byte, header.PointRecordLength)
	_, err := las.f.ReadAt(data, int64(offset))
	if err != nil {
		log.Fatalf("error while reading point at position: %v", err)
	}

	var x, y, z float64
	var r, g, b uint8
	var intensity uint8
	var classification uint8
	xyzOffsetValues := xyzOffets[header.PointFormatID]
	xOffset := xyzOffsetValues[0]
	yOffset := xyzOffsetValues[1]
	zOffset := xyzOffsetValues[2]
	x = float64(int32(binary.LittleEndian.Uint32(data[xOffset:xOffset+4])))*header.XScaleFactor + header.XOffset
	y = float64(int32(binary.LittleEndian.Uint32(data[yOffset:yOffset+4])))*header.YScaleFactor + header.YOffset
	z = float64(int32(binary.LittleEndian.Uint32(data[zOffset:zOffset+4])))*header.ZScaleFactor + header.ZOffset

	rgbOffsetValues := rgbOffets[header.PointFormatID]
	if rgbOffsetValues != nil {
		rOffset := rgbOffsetValues[0]
		gOffset := rgbOffsetValues[1]
		bOffset := rgbOffsetValues[2]
		var conversionFactor = uint16(256)
		if l.eightBitColor {
			conversionFactor = uint16(1)
		}

		r = uint8(binary.LittleEndian.Uint16(data[rOffset:rOffset+2]) / conversionFactor)
		g = uint8(binary.LittleEndian.Uint16(data[gOffset:gOffset+2]) / conversionFactor)
		b = uint8(binary.LittleEndian.Uint16(data[bOffset:bOffset+2]) / conversionFactor)
	}
	intensityOffset := 12
	intensity = uint8(binary.LittleEndian.Uint16(data[intensityOffset:intensityOffset+2]) / 256)
	classificationOffset := classificationOffets[header.PointFormatID]
	classification = data[classificationOffset]

	return &LasPoint{
		X:     x,
		Y:     y,
		Z:     z,
		R:     r,
		G:     g,
		B:     b,
		Int:   intensity,
		Class: classification,
	}
}
