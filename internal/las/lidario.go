package las

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
)

// NoData value used when indexing data outside of allowable range.
var NoData = math.Inf(-1)

// LasFile is a structure for manipulating LAS files.
type LasFile struct {
	fileName string
	f        *os.File
	Header   LasHeader
	VlrData  []VLR
	geokeys  GeoKeys
	sync.RWMutex
}

// Close closes a LasFile
func (las *LasFile) Close() error {
	if las.f == nil {
		// do nothing
		return errors.New("the LAS reader is nil")
	}
	return las.f.Close()
}

func (las *LasFile) readHeader() error {
	las.Lock()
	defer las.Unlock()
	b := make([]byte, 383)
	if _, err := las.f.ReadAt(b[0:383], 0); err != nil && err != io.EOF {
		return err
	}

	las.Header.projectIDUsed = true
	las.Header.VersionMajor = b[24]
	las.Header.VersionMinor = b[25]

	if las.Header.VersionMajor < 1 || las.Header.VersionMajor > 2 || las.Header.VersionMinor > 5 {
		// There's something wrong. It could be that the project ID values are not included in the header.
		las.Header.VersionMajor = b[8]
		las.Header.VersionMinor = b[9]
		if las.Header.VersionMajor < 1 || las.Header.VersionMajor > 2 || las.Header.VersionMinor > 5 {
			// There's something very wrong. Throw an error.
			return errors.New("either the file is formatted incorrectly or it is an unsupported LAS version")
		}
		las.Header.projectIDUsed = false
	}
	var offset uint
	las.Header.FileSignature = string(b[offset : offset+4])
	offset += 4
	las.Header.FileSourceID = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
	offset += 2
	las.Header.GlobalEncoding = GlobalEncodingField{binary.LittleEndian.Uint16(b[offset : offset+2])}
	offset += 2
	if las.Header.projectIDUsed {
		las.Header.ProjectID1 = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
		las.Header.ProjectID2 = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		offset += 2
		las.Header.ProjectID3 = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		offset += 2
		for i := 0; i < 8; i++ {
			las.Header.ProjectID4[i] = b[offset]
			offset++
		}
	}
	// The version major and minor are read earlier.
	// Two bytes must be added to the offset here.
	offset += 2
	las.Header.SystemID = string(b[offset : offset+32])
	las.Header.SystemID = strings.Trim(las.Header.SystemID, " ")
	las.Header.SystemID = strings.Trim(las.Header.SystemID, "\x00")
	offset += 32
	las.Header.GeneratingSoftware = string(b[offset : offset+32])
	las.Header.GeneratingSoftware = strings.Trim(las.Header.GeneratingSoftware, " ")
	las.Header.GeneratingSoftware = strings.Trim(las.Header.GeneratingSoftware, "\x00")
	offset += 32
	las.Header.FileCreationDay = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
	offset += 2
	las.Header.FileCreationYear = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
	offset += 2
	las.Header.HeaderSize = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
	offset += 2
	las.Header.OffsetToPoints = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	las.Header.NumberOfVLRs = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	las.Header.PointFormatID = b[104]
	offset++
	las.Header.PointRecordLength = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
	offset += 2
	// this might be zero, it's a legacy field in LAS 1.4
	las.Header.NumberPoints = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	for i := 0; i < 5; i++ {
		// this might be zero, it's a legacy field in LAS 1.4
		las.Header.NumberPointsByReturn[i] = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
	}

	las.Header.XScaleFactor = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.YScaleFactor = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.ZScaleFactor = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.XOffset = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.YOffset = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.ZOffset = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MaxX = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MinX = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MaxY = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MinY = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MaxZ = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	las.Header.MinZ = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
	offset += 8
	if las.Header.VersionMajor == 1 && (las.Header.VersionMinor == 3 || las.Header.VersionMinor == 4) {
		las.Header.WaveformDataStart = binary.LittleEndian.Uint64(b[offset : offset+8])
		offset += 8
	}
	if las.Header.VersionMajor == 1 && las.Header.VersionMinor == 4 {
		// Skip start of first Extended Variable Length and number of EVLR
		offset += 12
		// For Las 1.4 get the number of points from the new fields

		las.Header.NumberPoints = int(binary.LittleEndian.Uint32(b[offset : offset+8]))
		offset += 8
		for i := 0; i < 15; i++ {
			las.Header.NumberPointsByReturn[i] = int(binary.LittleEndian.Uint32(b[offset : offset+8]))
			offset += 8
		}
	}
	return nil
}

func (las *LasFile) readVLRs() error {
	las.Lock()
	defer las.Unlock()
	// Update the VLR slice
	las.VlrData = make([]VLR, las.Header.NumberOfVLRs)

	// Estimate how many bytes are used to store the VLRs
	vlrLength := las.Header.OffsetToPoints - las.Header.HeaderSize
	b := make([]byte, vlrLength)
	// if _, err := las.r.ReadAt(b[0:vlrLength], int64(las.Header.HeaderSize)); err != nil && err != io.EOF {
	if _, err := las.f.ReadAt(b, int64(las.Header.HeaderSize)); err != nil && err != io.EOF {
		return err
	}

	offset := 0
	for i := 0; i < las.Header.NumberOfVLRs; i++ {
		vlr := VLR{}
		vlr.Reserved = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		offset += 2
		vlr.UserID = string(b[offset : offset+16])
		vlr.UserID = strings.Trim(vlr.UserID, " ")
		vlr.UserID = strings.Trim(vlr.UserID, "\x00")
		offset += 16
		vlr.RecordID = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		offset += 2
		vlr.RecordLengthAfterHeader = int(binary.LittleEndian.Uint16(b[offset : offset+2]))
		offset += 2
		vlr.Description = string(b[offset : offset+32])
		vlr.Description = strings.Trim(vlr.Description, " ")
		vlr.Description = strings.Trim(vlr.Description, "\x00")
		offset += 32
		vlr.BinaryData = make([]uint8, vlr.RecordLengthAfterHeader)
		for j := 0; j < vlr.RecordLengthAfterHeader; j++ {
			// vlr.BinaryData = append(vlr.BinaryData, b[offset])
			vlr.BinaryData[j] = b[offset]
			offset++
		}
		if vlr.RecordID == 34735 {
			// GeoKey directory
			las.geokeys.addKeyDirectory(vlr.BinaryData)
		} else if vlr.RecordID == 34736 {
			// Double GeoKey parameters
			las.geokeys.addDoubleParams(vlr.BinaryData)
		} else if vlr.RecordID == 34737 {
			// ASCII GeoKey parameters
			las.geokeys.addASCIIParams(vlr.BinaryData)
		}
		las.VlrData[i] = vlr
	}

	return nil
}

// PrintGeokeys interprets the Geokeys, if there are any.
func (las *LasFile) PrintGeokeys() string {
	return las.geokeys.interpretGeokeys()
}

// LasHeader is a LAS file header structure.
type LasHeader struct {
	FileSignature        string
	FileSourceID         int
	GlobalEncoding       GlobalEncodingField
	ProjectID1           int
	ProjectID2           int
	ProjectID3           int
	ProjectID4           [8]byte
	VersionMajor         byte
	VersionMinor         byte
	SystemID             string // 32 characters
	GeneratingSoftware   string // 32 characters
	FileCreationDay      int
	FileCreationYear     int
	HeaderSize           int
	OffsetToPoints       int
	NumberOfVLRs         int
	PointFormatID        byte
	PointRecordLength    int
	NumberPoints         int
	NumberPointsByReturn [15]int
	XScaleFactor         float64
	YScaleFactor         float64
	ZScaleFactor         float64
	XOffset              float64
	YOffset              float64
	ZOffset              float64
	MaxX                 float64
	MinX                 float64
	MaxY                 float64
	MinY                 float64
	MaxZ                 float64
	MinZ                 float64
	WaveformDataStart    uint64
	projectIDUsed        bool
}

func (h LasHeader) String() string {
	var buffer bytes.Buffer
	// Buffer.WriteString("Las File Header:\n")

	s := fmt.Sprintf("File Signature: %v\n", h.FileSignature)
	buffer.WriteString(s)
	s = fmt.Sprintf("File Source ID: %v\n", h.FileSourceID)
	buffer.WriteString(s)
	s = fmt.Sprintf("Global Encoding: %v\n", h.GlobalEncoding)
	buffer.WriteString(s)

	if h.projectIDUsed {
		s = fmt.Sprintf("Project ID (GUID): %v-%v-%v-%v%v-%v%v%v%v%v%v\n", h.ProjectID1, h.ProjectID2, h.ProjectID3,
			h.ProjectID4[0], h.ProjectID4[1], h.ProjectID4[2], h.ProjectID4[3], h.ProjectID4[4], h.ProjectID4[5],
			h.ProjectID4[6], h.ProjectID4[7])
		buffer.WriteString(s)
	}

	s = fmt.Sprintf("System ID: %v\n", h.SystemID)
	buffer.WriteString(s)
	s = fmt.Sprintf("Generating Software: %v\n", h.GeneratingSoftware)
	buffer.WriteString(s)
	s = fmt.Sprintf("Las Version: %v.%v\n", h.VersionMajor, h.VersionMinor)
	buffer.WriteString(s)
	s = fmt.Sprintf("File Creation Day/Year: %v/%v\n", h.FileCreationDay, h.FileCreationYear)
	buffer.WriteString(s)
	s = fmt.Sprintf("Header Size: %v\n", h.HeaderSize)
	buffer.WriteString(s)
	s = fmt.Sprintf("Offset to points: %v\n", h.OffsetToPoints)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of VLRs: %v\n", h.NumberOfVLRs)
	buffer.WriteString(s)
	s = fmt.Sprintf("Point Format: %v\n", h.PointFormatID)
	buffer.WriteString(s)
	s = fmt.Sprintf("Point Record Length: %v\n", h.PointRecordLength)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of points: %v\n", h.NumberPoints)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of points by Return: [%v, %v, %v, %v, %v]\n", h.NumberPointsByReturn[0],
		h.NumberPointsByReturn[1], h.NumberPointsByReturn[2], h.NumberPointsByReturn[3],
		h.NumberPointsByReturn[4])
	buffer.WriteString(s)
	s = fmt.Sprintf("X Scale Factor: %f\n", h.XScaleFactor)
	buffer.WriteString(s)
	s = fmt.Sprintf("Y Scale Factor: %f\n", h.YScaleFactor)
	buffer.WriteString(s)
	s = fmt.Sprintf("Z Scale Factor: %f\n", h.ZScaleFactor)
	buffer.WriteString(s)
	s = fmt.Sprintf("X Offset: %f\n", h.XOffset)
	buffer.WriteString(s)
	s = fmt.Sprintf("Y Offset: %f\n", h.YOffset)
	buffer.WriteString(s)
	s = fmt.Sprintf("Z Offset: %f\n", h.ZOffset)
	buffer.WriteString(s)
	s = fmt.Sprintf("Max X: %f\n", h.MaxX)
	buffer.WriteString(s)
	s = fmt.Sprintf("Min X: %f\n", h.MinX)
	buffer.WriteString(s)
	s = fmt.Sprintf("Max Y: %f\n", h.MaxY)
	buffer.WriteString(s)
	s = fmt.Sprintf("Min Y: %f\n", h.MinY)
	buffer.WriteString(s)
	s = fmt.Sprintf("Max Z: %f\n", h.MaxZ)
	buffer.WriteString(s)
	s = fmt.Sprintf("Min Z: %f\n", h.MinZ)
	buffer.WriteString(s)
	s = fmt.Sprintf("Waveform Data Start: %v\n", h.WaveformDataStart)
	buffer.WriteString(s)

	return buffer.String()
}

// GlobalEncodingField contains the global encoding information in a LAS header
type GlobalEncodingField struct {
	Value uint16
}

// GpsTime returns the type of time format used in this file
func (gef GlobalEncodingField) GpsTime() GpsTimeType {
	if (gef.Value & 1) == 1 {
		return SatelliteGpsTime
	}
	return GpsWeekTime
}

// WaveformDataInternal returns a boolean indicating whether
// waveform packet data is stored internally to the file.
func (gef GlobalEncodingField) WaveformDataInternal() bool {
	return (gef.Value & 2) == 2
}

// WaveformDataExternal returns a boolean indicating whether
// waveform packet data is stored internally to the file.
func (gef GlobalEncodingField) WaveformDataExternal() bool {
	return (gef.Value & 4) == 4
}

// ReturnDataSynthetic returns a boolean indicating whether the
// return numbers have been generated synthetically.
func (gef GlobalEncodingField) ReturnDataSynthetic() bool {
	return (gef.Value & 8) == 8
}

// CoordinateReferenceSystemMethod returns the co-ordinate reference
// system method used within the file.
func (gef GlobalEncodingField) CoordinateReferenceSystemMethod() CoordinateReferenceSystemMethod {
	if (gef.Value & 16) == 16 {
		return WellKnownText
	}
	return GeoTiff
}

func (gef GlobalEncodingField) String() string {
	var buffer bytes.Buffer
	var str string
	str = fmt.Sprintf("\nGpsTime: %v\n", gef.GpsTime())
	buffer.WriteString(str)
	str = fmt.Sprintf("WaveformDataInternal: %v\n", gef.WaveformDataInternal())
	buffer.WriteString(str)
	str = fmt.Sprintf("WaveformDataExternal: %v\n", gef.WaveformDataExternal())
	buffer.WriteString(str)
	str = fmt.Sprintf("ReturnDataSynthetic: %v\n", gef.ReturnDataSynthetic())
	buffer.WriteString(str)
	str = fmt.Sprintf("CoordinateReferenceSystemMethod: %v", gef.CoordinateReferenceSystemMethod())
	buffer.WriteString(str)

	return buffer.String()
}

// GpsTimeType is a uint describing the type of time format used in
// in the file to store data GPS time.
type GpsTimeType uint

func (gtt GpsTimeType) String() string {
	if gtt == 1 {
		return "SatelliteGpsTime"
	}
	return "GpsWeekTime"
}

const (
	// SatelliteGpsTime represents Satellite GPS time
	SatelliteGpsTime = iota + 1
	// GpsWeekTime represents GPS week time
	GpsWeekTime
)

// CoordinateReferenceSystemMethod is the type of
// coordiante reference system used in the file, either
// Well-Known Text (WKT) or GeoTiff.
type CoordinateReferenceSystemMethod uint

func (crsm CoordinateReferenceSystemMethod) String() string {
	if crsm == 1 {
		return "WellKnownText"
	}
	return "GeoTiff"
}

const (
	// WellKnownText coordinate system reference method
	WellKnownText = iota + 1
	// GeoTiff coordinate system reference method
	GeoTiff
)

// VLR is a variable length record data structure
type VLR struct {
	Reserved                int
	UserID                  string // 16 characters
	RecordID                int
	RecordLengthAfterHeader int
	Description             string // 32 characters
	BinaryData              []uint8
}

func (vlr VLR) String() string {
	var buffer bytes.Buffer
	var str string
	str = fmt.Sprintf("\nReserved: %v\n", vlr.Reserved)
	buffer.WriteString(str)
	str = fmt.Sprintf("UserID: %v\n", vlr.UserID)
	buffer.WriteString(str)
	str = fmt.Sprintf("RecordID: %v\n", vlr.RecordID)
	buffer.WriteString(str)
	str = fmt.Sprintf("RecordLengthAfterHeader: %v\n", vlr.RecordLengthAfterHeader)
	buffer.WriteString(str)
	str = fmt.Sprintf("Description: %v", vlr.Description)
	buffer.WriteString(str)

	if vlr.RecordID == 34735 {
		// GeoKey directory
		buffer.WriteString("\nData: ")
		valSize := 2
		numVals := len(vlr.BinaryData) / valSize
		offset := 0
		for i := 0; i < numVals; i++ {
			val := binary.LittleEndian.Uint16(vlr.BinaryData[offset : offset+valSize])
			if i == 0 {
				str = fmt.Sprintf("[%v", val)
			} else if i == numVals-1 {
				str = fmt.Sprintf("%v]", val)
			} else {
				str = fmt.Sprintf(", %v", val)
			}
			buffer.WriteString(str)
			offset += valSize
		}
	} else if vlr.RecordID == 34736 {
		// Double GeoKey parameters
		buffer.WriteString("\nData: ")
		valSize := 8
		numVals := len(vlr.BinaryData) / valSize
		offset := 0
		for i := 0; i < numVals; i++ {
			val := math.Float64frombits(binary.LittleEndian.Uint64(vlr.BinaryData[offset : offset+valSize]))
			if i == 0 {
				str = fmt.Sprintf("[%f", val)
			} else if i == numVals-1 {
				str = fmt.Sprintf(", %f]", val)
			} else {
				str = fmt.Sprintf(", %f", val)
			}
			buffer.WriteString(str)
			offset += valSize
		}
	} else if vlr.RecordID == 34737 {
		// ASCII GeoKey parameters
		buffer.WriteString("\nData: ")
		str = string(vlr.BinaryData[0:])
		str = strings.Trim(str, " ")
		str = strings.Trim(str, "\x00")
		buffer.WriteString(str)
	} else {
		buffer.WriteString(fmt.Sprintf("\nBinaryData: [%v", vlr.BinaryData[0]))
		for i := 1; i < len(vlr.BinaryData); i++ {
			str = fmt.Sprintf(", %v", vlr.BinaryData[i])
			buffer.WriteString(str)
		}
		buffer.WriteString("]")
	}

	return buffer.String()
}
