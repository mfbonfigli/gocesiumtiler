package las

import (
	"encoding/binary"
	"log"
	"os"
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
	nil,          // Point format 0
	nil,          // Point format 1
	{20, 22, 24}, // Point format 2
	{28, 30, 32}, // Point format 3
	nil,          // Point format 4
	{28, 30, 32}, // Point format 5
	nil,          // Point format 6
	{30, 32, 34}, // Point format 7
	{30, 32, 34}, // Point format 8
	nil,          // Point format 9
	{30, 32, 34}, // Point format 10
}

// intensity offset is always 12

var classificationOffets = [11]int{
	15, // Point format 0
	15, // Point format 1
	15, // Point format 2
	15, // Point format 3
	15, // Point format 4
	15, // Point format 5
	16, // Point format 6
	16, // Point format 7
	16, // Point format 8
	16, // Point format 9
	16, // Point format 10
}

type LasReader interface {
	NumberOfPoints() int
	GetPointAt(int) (x, y, z float64, r, g, b, intensity, classification uint8)
	GetSrid() int
}

type FileLasReader struct {
	f             *LasFile
	eightBitColor bool
	srid          int
}

func NewFileLasReader(fileName string, srid int, eightBitColor bool) (*FileLasReader, error) {
	vlrs := []VLR{}
	las := LasFile{fileName: fileName, Header: LasHeader{}, VlrData: vlrs}
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
	return &FileLasReader{
		f:             &las,
		eightBitColor: eightBitColor,
		srid:          srid,
	}, nil
}

func (f *FileLasReader) NumberOfPoints() int {
	return f.f.Header.NumberPoints
}

func (f *FileLasReader) GetPointAt(i int) (x, y, z float64, r, g, b, intensity, classification uint8) {
	data := make([]byte, f.f.Header.PointRecordLength)
	if _, err := f.f.f.ReadAt(data, int64(f.f.Header.OffsetToPoints+i*f.f.Header.PointRecordLength)); err != nil {
		log.Fatalf("unable to read point n.%d: %v", i, err)
	}
	header := f.f.Header
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
		if f.eightBitColor {
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

	return x, y, z, r, g, b, intensity, classification
}

func (f *FileLasReader) GetSrid() int {
	return f.srid
}
