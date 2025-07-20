package golas

type ScanDirection uint8
type EdgeOfFlightLine uint8
type ClassificationFlag uint8

const (
	ScanDirectionNegative ScanDirection = 0
	ScanDirectionPositive ScanDirection = 1
)

const (
	EOFNormal    EdgeOfFlightLine = 0
	EOFEndOfScan EdgeOfFlightLine = 1
)

const (
	ClassificationSynthetic ClassificationFlag = 0
	ClassificationKeyPoint  ClassificationFlag = 1
	ClassificationWitheld   ClassificationFlag = 2
	ClassificationOverlap   ClassificationFlag = 3
)

// Point stores the raw data of a generic LAS Point. It is compatible with all LAS 1.1+ point formats,
// although some fields might be set at the default value for point formats that do not support them.
// The X, Y and Z coordinates stored are meant to be stored already scaled and offseted by the scale
// and offset values stored in the LAS header.
type Point struct {
	X                           float64
	Y                           float64
	Z                           float64
	Intensity                   uint16
	Classification              uint8
	ScanAngleRank               int8
	UserData                    uint8
	ScanAngle                   int16
	PointSourceID               uint16
	GPSTime                     float64
	Red                         uint16
	Green                       uint16
	Blue                        uint16
	NIR                         uint16
	WavePacketDescriptorIndex   uint8
	ByteOffsetToWaveformData    uint64
	WaveformPacketSizeBytes     uint32
	ReturnPointWaveformLocation float32
	ParametricDx                float32
	ParametricDy                float32
	ParametricDz                float32
	PointDataRecordFormat       uint8
	flags1                      byte
	flags2                      byte
	classificationRaw           byte
}

func (p Point) ReturnNumber() uint8 {
	if p.PointDataRecordFormat < 6 {
		return p.flags1 & 0b111
	}
	return p.flags1 & 0b1111
}

func (p Point) NumberOfReturns() uint8 {
	if p.PointDataRecordFormat < 6 {
		return (p.flags1 & 0b111000) >> 3
	}
	return (p.flags1 & 0b11110000) >> 4
}

func (p Point) ScanDirectionFlag() ScanDirection {
	if p.PointDataRecordFormat < 6 {
		return ScanDirection((p.flags1 & 0b1000000) >> 6)
	}
	return ScanDirection((p.flags2 & 0b1000000) >> 6)
}

func (p Point) EdgeOfFlightLineFlag() EdgeOfFlightLine {
	if p.PointDataRecordFormat < 6 {
		return EdgeOfFlightLine((p.flags1 & 0b10000000) >> 7)
	}
	return EdgeOfFlightLine((p.flags2 & 0b10000000) >> 7)
}

func (p Point) ScannerChannel() uint8 {
	if p.PointDataRecordFormat < 6 {
		return 0
	}
	return (p.flags2 & 0b110000) >> 4
}

func (p Point) ClassificationFlags() []ClassificationFlag {
	flags := []ClassificationFlag{}
	if p.PointDataRecordFormat < 6 {
		if (p.classificationRaw>>5)&0b1 == 1 {
			flags = append(flags, ClassificationSynthetic)
		}
		if (p.classificationRaw>>6)&0b1 == 1 {
			flags = append(flags, ClassificationKeyPoint)
		}
		if (p.classificationRaw>>7)&0b1 == 1 {
			flags = append(flags, ClassificationWitheld)
		}
		return flags
	}
	if (p.flags2 & 0b1) == 1 {
		flags = append(flags, ClassificationSynthetic)
	}
	if (p.flags2>>1)&0b1 == 1 {
		flags = append(flags, ClassificationKeyPoint)
	}
	if (p.flags2>>2)&0b1 == 1 {
		flags = append(flags, ClassificationWitheld)
	}
	if (p.flags2>>3)&0b1 == 1 {
		flags = append(flags, ClassificationOverlap)
	}
	return flags
}
