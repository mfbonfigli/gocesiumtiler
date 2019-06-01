package lidario

import (
	"bufio"
	"bytes"
	"cesium_tiler/structs/octree"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NoData value used when indexing point outside of allowable range.
var NoData = math.Inf(-1)

// LasFile is a structure for manipulating LAS files.
type LasFile struct {
	fileName               string
	fileMode               string
	f                      *os.File
	Header                 LasHeader
	VlrData                []VLR
	geokeys                GeoKeys
	pointData              []PointRecord0
	pointDataOctElement    []octree.OctElement
	gpsData                []float64
	rgbData                []RgbData
	usePointIntensity      bool
	usePointUserdata       bool
	headerIsSet            bool
	fixedRadiusSearch2DSet bool
	frs2D                  *fixedRadiusSearch
	fixedRadiusSearch3DSet bool
	frs3D                  *fixedRadiusSearch
	sync.RWMutex
}

// NewLasFile creates a new LasFile structure.
func NewLasFile(fileName, fileMode string) (*LasFile, error) {
	fileMode = strings.ToLower(fileMode)
	// initialize the VLR array
	vlrs := []VLR{}
	las := LasFile{fileName: fileName, fileMode: fileMode, Header: LasHeader{}, VlrData: vlrs}
	if las.fileMode == "r" || las.fileMode == "rh" {
		if err := las.read(); err != nil {
			return &las, err
		}
	} else {
		las.fileMode = "w"
		fmt.Println("Okay, write the new file: ", fileName)
		var err error
		if las.f, err = os.Create(las.fileName); err != nil {
			return &las, err
		}

		// initialize the point, gps, and rgb data slices and set the capacity
		initCapacity := 1000000
		las.pointData = make([]PointRecord0, 0, initCapacity)
		if las.Header.PointFormatID == 1 || las.Header.PointFormatID == 3 {
			las.gpsData = make([]float64, 0, initCapacity)
		}

		if las.Header.PointFormatID == 2 || las.Header.PointFormatID == 3 {
			las.rgbData = make([]RgbData, 0, initCapacity)
		}
	}
	return &las, nil
}

// InitializeUsingFile initializes a new LAS file based on another existing file.
// The function transfers values from the header and the VLRs to the new file.
func InitializeUsingFile(fileName string, other *LasFile) (*LasFile, error) {
	las := LasFile{}
	las.fileName = fileName
	las.fileMode = "w"
	las.usePointIntensity = true
	las.usePointUserdata = true

	var err error
	if las.f, err = os.Create(las.fileName); err != nil {
		return &las, err
	}

	las.AddHeader(other.Header)

	// Copy the VLRs
	for _, vlr := range other.VlrData {
		las.AddVLR(vlr)
	}

	// initialize the point, gps, and rgb data slices and set the capacity to that of the other file
	las.pointData = make([]PointRecord0, 0, other.Header.NumberPoints)
	if other.Header.PointFormatID == 1 || other.Header.PointFormatID == 3 {
		las.gpsData = make([]float64, 0, other.Header.NumberPoints)
	}

	if other.Header.PointFormatID == 2 || other.Header.PointFormatID == 3 {
		las.rgbData = make([]RgbData, 0, other.Header.NumberPoints)
	}

	return &las, nil
}

// AddHeader adds a header to a LasFile created in 'w' (write) mode. The method is thread-safe.
func (las *LasFile) AddHeader(header LasHeader) error {
	las.Lock()
	// defer las.Unlock()
	if las.fileMode == "r" || las.fileMode == "rh" {
		las.Unlock()
		return fmt.Errorf("file has been opened in %v mode; AddHeader can only be used in 'w' mode", las.fileMode)
	}
	las.Header = header
	las.Header.NumberOfVLRs = 0
	las.Header.NumberPoints = 0
	las.Header.VersionMajor = 1
	las.Header.VersionMinor = 3

	// These must be set by the data
	las.Header.MinX = math.Inf(0)
	las.Header.MaxX = math.Inf(-1)
	las.Header.MinY = math.Inf(0)
	las.Header.MaxY = math.Inf(-1)
	las.Header.MinZ = math.Inf(0)
	las.Header.MaxZ = math.Inf(-1)

	las.Header.SystemID = fixedLengthString("GoSpatial by John Lindsay", 32)
	las.Header.GeneratingSoftware = fixedLengthString("GoSpatial by John Lindsay", 32)

	las.Header.XScaleFactor = 0.0001
	las.Header.YScaleFactor = 0.0001
	las.Header.ZScaleFactor = 0.0001

	las.headerIsSet = true

	las.Unlock()
	return nil
}

// AddVLR adds a variable length record (VLR) to a LAS file created in 'w' (write) mode. The method is thread-safe.
func (las *LasFile) AddVLR(vlr VLR) error {
	las.Lock()
	// defer las.Unlock()
	if las.fileMode == "r" || las.fileMode == "rh" {
		las.Unlock()
		return fmt.Errorf("file has been opened in %v mode; AddHeader can only be used in 'w' mode", las.fileMode)
	}
	// The header must be set before you can add VLRs
	if !las.headerIsSet {
		las.Unlock()
		return errors.New("the header of a LAS file must be added before any VLRs; Please see AddHeader()")
	}
	las.VlrData = append(las.VlrData, vlr)
	las.Header.NumberOfVLRs++
	las.Unlock()
	return nil
}

// AddLasPoint adds a point record to a Las file created in 'w' (write) mode. The method is thread-safe.
func (las *LasFile) AddLasPoint(p LasPointer) error {
	if las.fileMode == "r" || las.fileMode == "rh" {
		return fmt.Errorf("file has been opened in %v mode; AddHeader can only be used in 'w' mode", las.fileMode)
	}
	// The header must be set before you can add points
	if !las.headerIsSet {
		return errors.New("the header of a LAS file must be added before any points; Please see AddHeader()")
	}
	las.Lock()
	// defer las.Unlock()
	pd := p.PointData()
	las.pointData = append(las.pointData, *pd)

	switch p.Format() {
	case 1:
		las.gpsData = append(las.gpsData, p.GpsTimeData())
	case 2:
		las.rgbData = append(las.rgbData, *p.RgbData())
	case 3:
		las.gpsData = append(las.gpsData, p.GpsTimeData())
		las.rgbData = append(las.rgbData, *p.RgbData())
	default:
		// do nothing
	}

	val := pd.X
	if val < las.Header.MinX {
		las.Header.MinX = val
	}
	if val > las.Header.MaxX {
		las.Header.MaxX = val
	}

	val = pd.Y
	if val < las.Header.MinY {
		las.Header.MinY = val
	}
	if val > las.Header.MaxY {
		las.Header.MaxY = val
	}

	val = pd.Z
	if val < las.Header.MinZ {
		las.Header.MinZ = val
	}
	if val > las.Header.MaxZ {
		las.Header.MaxZ = val
	}

	whichReturn := pd.BitField.ReturnNumber()
	if whichReturn == 0 {
		whichReturn = 1
	}
	if whichReturn > 5 {
		whichReturn = 5
	}
	las.Header.NumberPointsByReturn[whichReturn-1]++
	las.Header.NumberPoints++
	las.Unlock()
	return nil
}

// AddLasPoints adds a slice of point record to a Las file created in 'w' (write) mode. The method is thread-safe.
func (las *LasFile) AddLasPoints(points []LasPointer) error {
	if las.fileMode == "r" || las.fileMode == "rh" {
		return fmt.Errorf("file has been opened in %v mode; AddHeader can only be used in 'w' mode", las.fileMode)
	}
	// The header must be set before you can add points
	if !las.headerIsSet {
		return errors.New("the header of a LAS file must be added before any points; Please see AddHeader()")
	}
	las.Lock()
	// defer las.Unlock()
	var pd PointRecord0
	var val float64
	var whichReturn uint8
	for _, p := range points {
		pd = *p.PointData()
		las.pointData = append(las.pointData, pd)

		// if p.Format() == 1 || p.Format() == 3 {
		// 	las.gpsData = append(las.gpsData, p.GpsTimeData())
		// }

		// if p.Format() == 2 || p.Format() == 3 {
		// 	las.rgbData = append(las.rgbData, *p.RgbData())
		// }

		switch p.Format() {
		case 1:
			las.gpsData = append(las.gpsData, p.GpsTimeData())
		case 2:
			las.rgbData = append(las.rgbData, *p.RgbData())
		case 3:
			las.gpsData = append(las.gpsData, p.GpsTimeData())
			las.rgbData = append(las.rgbData, *p.RgbData())
		default:
			// do nothing
		}

		val = pd.X
		if val < las.Header.MinX {
			las.Header.MinX = val
		}
		if val > las.Header.MaxX {
			las.Header.MaxX = val
		}

		val = pd.Y
		if val < las.Header.MinY {
			las.Header.MinY = val
		}
		if val > las.Header.MaxY {
			las.Header.MaxY = val
		}

		val = pd.Z
		if val < las.Header.MinZ {
			las.Header.MinZ = val
		}
		if val > las.Header.MaxZ {
			las.Header.MaxZ = val
		}

		whichReturn = pd.BitField.ReturnNumber()
		if whichReturn == 0 {
			whichReturn = 1
		}
		if whichReturn > 5 {
			whichReturn = 5
		}
		las.Header.NumberPointsByReturn[whichReturn-1]++
		las.Header.NumberPoints++
	}
	las.Unlock()
	return nil
}

// Close closes a LasFile
func (las *LasFile) Close() error {
	if las.f == nil {
		// do nothing
		return errors.New("the LAS reader is nil")
	}
	if las.fileMode == "w" {
		las.write()
	}
	return las.f.Close()
}

// GetXYZ returns the x, y, z data for a specified point
func (las *LasFile) GetXYZ(index int) (float64, float64, float64, error) {
	if index < 0 || index >= las.Header.NumberPoints {
		return NoData, NoData, NoData, errors.New("Index outside of allowable range")
	}
	return las.pointData[index].X, las.pointData[index].Y, las.pointData[index].Z, nil
}

// LasPoint returns a LAS point.
func (las *LasFile) LasPoint(index int) (LasPointer, error) {
	if index < 0 || index >= las.Header.NumberPoints {
		return &PointRecord0{}, errors.New("Index outside of allowable range")
	}
	if las.fileMode == "rh" {
		return &PointRecord0{}, errors.New("The file was opened in 'rh' (read header); data points were therefore not read from the file")
	}
	// las.RLock()
	// defer las.RUnlock()
	switch las.Header.PointFormatID {
	case 0:
		// las.RUnlock()
		return &las.pointData[index], nil
	case 1:
		// las.RUnlock()
		return &PointRecord1{PointRecord0: &las.pointData[index], GPSTime: las.gpsData[index]}, nil
	case 2:
		// las.RUnlock()
		return &PointRecord2{PointRecord0: &las.pointData[index], RGB: &las.rgbData[index]}, nil
	case 3:
		// las.RUnlock()
		return &PointRecord3{PointRecord0: &las.pointData[index], GPSTime: las.gpsData[index], RGB: &las.rgbData[index]}, nil
	default:
		// las.RUnlock()
		return &PointRecord0{}, errors.New("Unrecognized point format")
	}
}

func (las *LasFile) read() error {
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
		recLengths := [4][4]int{{20, 18, 19, 17}, {28, 26, 27, 25}, {26, 24, 25, 23}, {34, 32, 33, 31}}

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

		if err := las.readPoints(); err != nil {
			return err
		}
	}
	return nil
}

func (las *LasFile) readHeader() error {
	las.Lock()
	defer las.Unlock()
	b := make([]byte, 243)
	if _, err := las.f.ReadAt(b[0:243], 0); err != nil && err != io.EOF {
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
	las.Header.NumberPoints = int(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	for i := 0; i < 5; i++ {
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
	if las.Header.VersionMajor == 1 && las.Header.VersionMinor == 3 {
		las.Header.WaveformDataStart = binary.LittleEndian.Uint64(b[offset : offset+8])
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

func (las *LasFile) readPoints() error {
	las.Lock()
	defer las.Unlock()
	las.pointData = make([]PointRecord0, las.Header.NumberPoints)
	if las.Header.PointFormatID == 1 || las.Header.PointFormatID == 3 {
		las.gpsData = make([]float64, las.Header.NumberPoints)
	}
	if las.Header.PointFormatID == 2 || las.Header.PointFormatID == 3 {
		las.rgbData = make([]RgbData, las.Header.NumberPoints)
	}

	// Estimate how many bytes are used to store the points
	pointsLength := las.Header.NumberPoints * las.Header.PointRecordLength
	b := make([]byte, pointsLength)
	if _, err := las.f.ReadAt(b, int64(las.Header.OffsetToPoints)); err != nil && err != io.EOF {
		return err
	}

	// Intensity and userdata are both optional. Figure out if they need to be read.
	// The only way to do this is to compare the point record length by point format
	recLengths := [4][4]int{{20, 18, 19, 17}, {28, 26, 27, 25}, {26, 24, 25, 23}, {34, 32, 33, 31}}

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
			var p PointRecord0
			for i := pointSt; i <= pointEnd; i++ {
				offset = i * las.Header.PointRecordLength
				// p := PointRecord0{}
				p.X = float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.XScaleFactor + las.Header.XOffset
				offset += 4
				p.Y = float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.YScaleFactor + las.Header.YOffset
				offset += 4
				p.Z = float64(int32(binary.LittleEndian.Uint32(b[offset:offset+4])))*las.Header.ZScaleFactor + las.Header.ZOffset
				offset += 4
				if las.usePointIntensity {
					p.Intensity = binary.LittleEndian.Uint16(b[offset : offset+2])
					offset += 2
				}
				p.BitField = PointBitField{Value: b[offset]}
				offset++
				p.ClassBitField = ClassificationBitField{Value: b[offset]}
				offset++
				p.ScanAngle = int8(b[offset])
				offset++
				if las.usePointUserdata {
					p.UserData = b[offset]
					offset++
				}
				p.PointSourceID = binary.LittleEndian.Uint16(b[offset : offset+2])
				offset += 2

				las.pointData[i] = p

				if las.Header.PointFormatID == 1 || las.Header.PointFormatID == 3 {
					las.gpsData[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[offset : offset+8]))
					offset += 8
				}
				if las.Header.PointFormatID == 2 || las.Header.PointFormatID == 3 {
					rgb := RgbData{}
					rgb.Red = binary.LittleEndian.Uint16(b[offset : offset+2])
					offset += 2
					rgb.Green = binary.LittleEndian.Uint16(b[offset : offset+2])
					offset += 2
					rgb.Blue = binary.LittleEndian.Uint16(b[offset : offset+2])
					offset += 2
					las.rgbData[i] = rgb
				}
			}
		}(startingPoint, endingPoint)
		startingPoint = endingPoint + 1
	}

	wg.Wait()

	return nil
}

func (las *LasFile) write() error {
	las.Lock()
	defer las.Unlock()
	if las.fileMode == "r" || las.fileMode == "rh" {
		return fmt.Errorf("file has been opened in %v mode; AddHeader can only be used in 'w' mode", las.fileMode)
	}
	// The header must be set before you can add VLRs
	if !las.headerIsSet {
		return errors.New("the header of a LAS file must be added before you can write the file; Please see AddHeader()")
	}
	if las.Header.NumberPoints == 0 || len(las.pointData) != las.Header.NumberPoints {
		return errors.New("cannot write LAS file until points have been added; Please see AddLasPoint()")
	}

	las.Header.XOffset = las.Header.MinX
	las.Header.YOffset = las.Header.MinY
	las.Header.ZOffset = las.Header.MinZ

	mantissa := len(fmt.Sprintf("%v", math.Floor(las.Header.MaxX-las.Header.MinX)))
	dec := 1.0 / math.Pow10(8-mantissa)
	if las.Header.XScaleFactor == 0.0 {
		las.Header.XScaleFactor = dec
	}

	mantissa = len(fmt.Sprintf("%v", math.Floor(las.Header.MaxY-las.Header.MinY)))
	dec = 1.0 / math.Pow10(8-mantissa)
	if las.Header.YScaleFactor == 0.0 {
		las.Header.YScaleFactor = dec
	}

	mantissa = len(fmt.Sprintf("%v", math.Floor(las.Header.MaxZ-las.Header.MinZ)))
	dec = 1.0 / math.Pow10(8-mantissa)
	if las.Header.ZScaleFactor == 0.0 {
		las.Header.ZScaleFactor = dec
	}

	var err error

	if las.f == nil {
		if las.f, err = os.Create(las.fileName); err != nil {
			return err
		}
	}

	w := bufio.NewWriter(las.f)
	bytes2 := make([]byte, 2)
	bytes4 := make([]byte, 4)
	bytes8 := make([]byte, 8)

	//////////////////////////////////
	// Write the header to the file //
	//////////////////////////////////

	las.Header.FileSignature = "LASF"
	w.WriteString(las.Header.FileSignature)

	binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.FileSourceID))
	w.Write(bytes2)

	binary.LittleEndian.PutUint16(bytes2, las.Header.GlobalEncoding.Value)
	w.Write(bytes2)

	if las.Header.projectIDUsed {
		binary.LittleEndian.PutUint32(bytes4, uint32(las.Header.ProjectID1))
		w.Write(bytes4)
		binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.ProjectID2))
		w.Write(bytes2)
		binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.ProjectID3))
		w.Write(bytes2)
		w.Write(las.Header.ProjectID4[:])
	}

	las.Header.VersionMajor = 1
	w.WriteByte(las.Header.VersionMajor)
	las.Header.VersionMinor = 3
	w.WriteByte(las.Header.VersionMinor)

	if len(las.Header.SystemID) == 0 {
		las.Header.SystemID = fixedLengthString("OTHER", 32)
	} else {
		las.Header.SystemID = fixedLengthString(las.Header.SystemID, 32)
	}
	w.WriteString(las.Header.SystemID)

	las.Header.GeneratingSoftware = fixedLengthString("GoSpatial by John Lindsay", 32)
	w.WriteString(las.Header.GeneratingSoftware)

	t := time.Now()
	las.Header.FileCreationDay = t.YearDay()
	binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.FileCreationDay))
	w.Write(bytes2)
	las.Header.FileCreationYear = t.Year()
	binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.FileCreationYear))
	w.Write(bytes2)

	las.Header.HeaderSize = 235
	binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.HeaderSize))
	w.Write(bytes2)

	// Figure out the offset to the points
	totalVLRSize := 54 * las.Header.NumberOfVLRs
	for i := 0; i < las.Header.NumberOfVLRs; i++ {
		totalVLRSize += las.VlrData[i].RecordLengthAfterHeader
	}
	las.Header.OffsetToPoints = 235 + totalVLRSize
	binary.LittleEndian.PutUint32(bytes4, uint32(las.Header.OffsetToPoints))
	w.Write(bytes4)

	binary.LittleEndian.PutUint32(bytes4, uint32(las.Header.NumberOfVLRs))
	w.Write(bytes4)

	w.WriteByte(las.Header.PointFormatID)

	// Intensity and userdata are both optional. Figure out if they need to be read.
	// The only way to do this is to compare the point record length by point format
	recLengths := [][]int{{20, 18, 19, 17}, {28, 26, 27, 25}, {26, 24, 25, 23}, {34, 32, 33, 31}}

	if las.usePointIntensity && las.usePointUserdata {
		las.Header.PointRecordLength = recLengths[las.Header.PointFormatID][0]
	} else if !las.usePointIntensity && las.usePointUserdata {
		las.Header.PointRecordLength = recLengths[las.Header.PointFormatID][1]
	} else if las.usePointIntensity && !las.usePointUserdata {
		las.Header.PointRecordLength = recLengths[las.Header.PointFormatID][2]
	} else { //if !las.usePointIntensity && !las.usePointUserdata {
		las.Header.PointRecordLength = recLengths[las.Header.PointFormatID][3]
	}

	binary.LittleEndian.PutUint16(bytes2, uint16(las.Header.PointRecordLength))
	w.Write(bytes2)

	binary.LittleEndian.PutUint32(bytes4, uint32(las.Header.NumberPoints))
	w.Write(bytes4)

	for i := 0; i < 5; i++ {
		binary.LittleEndian.PutUint32(bytes4, uint32(las.Header.NumberPointsByReturn[i]))
		w.Write(bytes4)
	}

	bits := math.Float64bits(las.Header.XScaleFactor)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.YScaleFactor)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.ZScaleFactor)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.XOffset)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.YOffset)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.ZOffset)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MaxX)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MinX)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MaxY)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MinY)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MaxZ)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	bits = math.Float64bits(las.Header.MinZ)
	binary.LittleEndian.PutUint64(bytes8, bits)
	w.Write(bytes8)

	binary.LittleEndian.PutUint64(bytes8, las.Header.WaveformDataStart)
	w.Write(bytes8)

	////////////////////////////////
	// Write the VLRs to the file //
	////////////////////////////////
	for i := 0; i < las.Header.NumberOfVLRs; i++ {
		vlr := las.VlrData[i]

		binary.LittleEndian.PutUint16(bytes2, uint16(vlr.Reserved))
		w.Write(bytes2)

		w.WriteString(fixedLengthString(vlr.UserID, 16))

		binary.LittleEndian.PutUint16(bytes2, uint16(vlr.RecordID))
		w.Write(bytes2)

		binary.LittleEndian.PutUint16(bytes2, uint16(vlr.RecordLengthAfterHeader))
		w.Write(bytes2)

		w.WriteString(fixedLengthString(vlr.Description, 32))

		w.Write(vlr.BinaryData)
	}

	//////////////////////////////////
	// Write the points to the file //
	//////////////////////////////////
	numCPUs := runtime.NumCPU()
	var wg sync.WaitGroup
	blockSize := las.Header.NumberPoints / numCPUs
	var startingPoint int

	// how many bytes will it take; create a byte slice of the appropriate length
	b := make([]byte, las.Header.NumberPoints*las.Header.PointRecordLength)

	switch las.Header.PointFormatID {
	case 0:
		for startingPoint < las.Header.NumberPoints {
			endingPoint := startingPoint + blockSize
			if endingPoint >= las.Header.NumberPoints {
				endingPoint = las.Header.NumberPoints - 1
			}
			wg.Add(1)
			go func(pointSt, pointEnd int) {
				defer wg.Done()
				var val int32
				var offset int
				var p PointRecord0
				b2 := make([]byte, 2)
				b4 := make([]byte, 4)
				for i := pointSt; i <= pointEnd; i++ {
					p = las.pointData[i]

					offset = i * las.Header.PointRecordLength

					val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					if las.usePointIntensity {
						binary.LittleEndian.PutUint16(b2, p.Intensity)
						b[offset] = b2[0]
						b[offset+1] = b2[1]
						offset += 2
					}

					b[offset] = p.BitField.Value
					b[offset+1] = p.ClassBitField.Value
					b[offset+2] = uint8(p.ScanAngle)
					offset += 3

					if las.usePointUserdata {
						b[offset] = p.UserData
						offset++
					}

					binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					// buf := new(bytes.Buffer)

					// val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// if las.usePointIntensity {
					// 	binary.LittleEndian.PutUint16(b2, p.Intensity)
					// 	buf.Write(b2)
					// }

					// buf.WriteByte(p.BitField.Value)
					// buf.WriteByte(p.ClassBitField.Value)
					// buf.WriteByte(uint8(p.ScanAngle))

					// if las.usePointUserdata {
					// 	buf.WriteByte(p.UserData)
					// }

					// binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					// buf.Write(b2)

					// d := buf.Bytes()
					// offset := i * uint32(las.Header.PointRecordLength)
					// for j := 0; j < len(d); j++ {
					// 	b[offset+uint32(j)] = d[j]
					// }
				}

			}(startingPoint, endingPoint)
			startingPoint = endingPoint + 1
		}

	case 1:
		for startingPoint < las.Header.NumberPoints {
			endingPoint := startingPoint + blockSize
			if endingPoint >= las.Header.NumberPoints {
				endingPoint = las.Header.NumberPoints - 1
			}
			wg.Add(1)
			go func(pointSt, pointEnd int) {
				defer wg.Done()
				var val int32
				var p PointRecord0
				var bits uint64
				var offset int
				b2 := make([]byte, 2)
				b4 := make([]byte, 4)
				b8 := make([]byte, 8)
				for i := pointSt; i <= pointEnd; i++ {
					p = las.pointData[i]

					offset = i * las.Header.PointRecordLength

					val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					if las.usePointIntensity {
						binary.LittleEndian.PutUint16(b2, p.Intensity)
						b[offset] = b2[0]
						b[offset+1] = b2[1]
						offset += 2
					}

					b[offset] = p.BitField.Value
					b[offset+1] = p.ClassBitField.Value
					b[offset+2] = uint8(p.ScanAngle)
					offset += 3

					if las.usePointUserdata {
						b[offset] = p.UserData
						offset++
					}

					binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					bits = math.Float64bits(las.gpsData[i])
					binary.LittleEndian.PutUint64(b8, bits)
					b[offset] = b8[0]
					b[offset+1] = b8[1]
					b[offset+2] = b8[2]
					b[offset+3] = b8[3]
					b[offset+4] = b8[4]
					b[offset+5] = b8[5]
					b[offset+6] = b8[6]
					b[offset+7] = b8[7]
					offset += 8

					// buf := new(bytes.Buffer)

					// val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// if las.usePointIntensity {
					// 	binary.LittleEndian.PutUint16(b2, p.Intensity)
					// 	buf.Write(b2)
					// }

					// buf.WriteByte(p.BitField.Value)
					// buf.WriteByte(p.ClassBitField.Value)
					// buf.WriteByte(uint8(p.ScanAngle))

					// if las.usePointUserdata {
					// 	buf.WriteByte(p.UserData)
					// }

					// binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					// buf.Write(b2)

					// bits := math.Float64bits(las.gpsData[i])
					// binary.LittleEndian.PutUint64(b8, bits)
					// buf.Write(b8)

					// d := buf.Bytes()
					// offset := i * uint32(las.Header.PointRecordLength)
					// for j := 0; j < len(d); j++ {
					// 	b[offset+uint32(j)] = d[j]
					// }
				}

			}(startingPoint, endingPoint)
			startingPoint = endingPoint + 1
		}

	case 2:
		for startingPoint < las.Header.NumberPoints {
			endingPoint := startingPoint + blockSize
			if endingPoint >= las.Header.NumberPoints {
				endingPoint = las.Header.NumberPoints - 1
			}
			wg.Add(1)
			go func(pointSt, pointEnd int) {
				defer wg.Done()
				var val int32
				var offset int
				var p PointRecord0
				b2 := make([]byte, 2)
				b4 := make([]byte, 4)
				for i := pointSt; i <= pointEnd; i++ {
					p = las.pointData[i]

					offset = i * las.Header.PointRecordLength

					val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					if las.usePointIntensity {
						binary.LittleEndian.PutUint16(b2, p.Intensity)
						b[offset] = b2[0]
						b[offset+1] = b2[1]
						offset += 2
					}

					b[offset] = p.BitField.Value
					b[offset+1] = p.ClassBitField.Value
					b[offset+2] = uint8(p.ScanAngle)
					offset += 3

					if las.usePointUserdata {
						b[offset] = p.UserData
						offset++
					}

					binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Red)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2
					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Green)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2
					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Blue)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					// p = las.pointData[i]
					// buf := new(bytes.Buffer)

					// val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// if las.usePointIntensity {
					// 	binary.LittleEndian.PutUint16(b2, p.Intensity)
					// 	buf.Write(b2)
					// }

					// buf.WriteByte(p.BitField.Value)
					// buf.WriteByte(p.ClassBitField.Value)
					// buf.WriteByte(uint8(p.ScanAngle))

					// if las.usePointUserdata {
					// 	buf.WriteByte(p.UserData)
					// }

					// binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					// buf.Write(b2)

					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Red)
					// buf.Write(b2)
					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Green)
					// buf.Write(b2)
					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Blue)
					// buf.Write(b2)

					// d := buf.Bytes()
					// offset := i * uint32(las.Header.PointRecordLength)
					// for j := uint32(0); j < uint32(len(d)); j++ {
					// 	b[offset+j] = d[j]
					// }
				}

			}(startingPoint, endingPoint)
			startingPoint = endingPoint + 1
		}

	case 3:
		for startingPoint < las.Header.NumberPoints {
			endingPoint := startingPoint + blockSize
			if endingPoint >= las.Header.NumberPoints {
				endingPoint = las.Header.NumberPoints - 1
			}
			wg.Add(1)
			go func(pointSt, pointEnd int) {
				defer wg.Done()
				var val int32
				var p PointRecord0
				var bits uint64
				var offset int
				b2 := make([]byte, 2)
				b4 := make([]byte, 4)
				b8 := make([]byte, 8)
				for i := pointSt; i <= pointEnd; i++ {
					p = las.pointData[i]

					offset = i * las.Header.PointRecordLength

					val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					binary.LittleEndian.PutUint32(b4, uint32(val))
					b[offset] = b4[0]
					b[offset+1] = b4[1]
					b[offset+2] = b4[2]
					b[offset+3] = b4[3]
					offset += 4

					if las.usePointIntensity {
						binary.LittleEndian.PutUint16(b2, p.Intensity)
						b[offset] = b2[0]
						b[offset+1] = b2[1]
						offset += 2
					}

					b[offset] = p.BitField.Value
					b[offset+1] = p.ClassBitField.Value
					b[offset+2] = uint8(p.ScanAngle)
					offset += 3

					if las.usePointUserdata {
						b[offset] = p.UserData
						offset++
					}

					binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					bits = math.Float64bits(las.gpsData[i])
					binary.LittleEndian.PutUint64(b8, bits)
					b[offset] = b8[0]
					b[offset+1] = b8[1]
					b[offset+2] = b8[2]
					b[offset+3] = b8[3]
					b[offset+4] = b8[4]
					b[offset+5] = b8[5]
					b[offset+6] = b8[6]
					b[offset+7] = b8[7]
					offset += 8

					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Red)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2
					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Green)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2
					binary.LittleEndian.PutUint16(b2, las.rgbData[i].Blue)
					b[offset] = b2[0]
					b[offset+1] = b2[1]
					offset += 2

					// buf := new(bytes.Buffer)

					// val = int32((p.X - las.Header.XOffset) / las.Header.XScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Y - las.Header.YOffset) / las.Header.YScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// val = int32((p.Z - las.Header.ZOffset) / las.Header.ZScaleFactor)
					// binary.LittleEndian.PutUint32(b4, uint32(val))
					// buf.Write(b4)

					// if las.usePointIntensity {
					// 	binary.LittleEndian.PutUint16(b2, p.Intensity)
					// 	buf.Write(b2)
					// }

					// buf.WriteByte(p.BitField.Value)
					// buf.WriteByte(p.ClassBitField.Value)
					// buf.WriteByte(uint8(p.ScanAngle))

					// if las.usePointUserdata {
					// 	buf.WriteByte(p.UserData)
					// }

					// binary.LittleEndian.PutUint16(b2, p.PointSourceID)
					// buf.Write(b2)

					// bits := math.Float64bits(las.gpsData[i])
					// binary.LittleEndian.PutUint64(b8, bits)
					// buf.Write(b8)

					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Red)
					// buf.Write(b2)
					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Green)
					// buf.Write(b2)
					// binary.LittleEndian.PutUint16(b2, las.rgbData[i].Blue)
					// buf.Write(b2)

					// d := buf.Bytes()
					// offset := i * uint32(las.Header.PointRecordLength)
					// for j := 0; j < len(d); j++ {
					// 	b[offset+uint32(j)] = d[j]
					// }
				}

			}(startingPoint, endingPoint)
			startingPoint = endingPoint + 1
		}
	}

	wg.Wait()
	w.Write(b)
	w.Flush()

	return nil
}

// FixedRadiusSearch2D performs a 2D fixed radius search
func (las *LasFile) FixedRadiusSearch2D(x, y float64) *FRSResultList { //[]FixedRadiusSearchResult {
	if !las.fixedRadiusSearch2DSet {
		panic("SetFixedRadiusSearch must be called with threeDimensionalSearch set to 'false' before performing a 2D search")
	}
	return las.frs2D.search2D(x, y)
}

// FixedRadiusSearch3D performs a 3D fixed radius search
func (las *LasFile) FixedRadiusSearch3D(x, y, z float64) *FRSResultList { //[]FixedRadiusSearchResult {
	if !las.fixedRadiusSearch3DSet {
		panic("SetFixedRadiusSearch must be called with threeDimensionalSearch set to 'true' before performing a 3D search")
	}
	return las.frs3D.search3D(x, y, z)
}

// SetFixedRadiusSearchDistance sets the fixed radius search
func (las *LasFile) SetFixedRadiusSearchDistance(radius float64, threeDimensionalSearch bool) error {
	if threeDimensionalSearch {
		las.frs3D = build(las, radius, threeDimensionalSearch)
		las.fixedRadiusSearch3DSet = true
	} else {
		las.frs2D = build(las, radius, threeDimensionalSearch)
		las.fixedRadiusSearch2DSet = true
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
	NumberPointsByReturn [5]int
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
	// buffer.WriteString("Las File Header:\n")

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
	s = fmt.Sprintf("Offset to Points: %v\n", h.OffsetToPoints)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of VLRs: %v\n", h.NumberOfVLRs)
	buffer.WriteString(s)
	s = fmt.Sprintf("Point Format: %v\n", h.PointFormatID)
	buffer.WriteString(s)
	s = fmt.Sprintf("Point Record Length: %v\n", h.PointRecordLength)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of Points: %v\n", h.NumberPoints)
	buffer.WriteString(s)
	s = fmt.Sprintf("Number of Points by Return: [%v, %v, %v, %v, %v]\n", h.NumberPointsByReturn[0],
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
// in the file to store point GPS time.
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

// LasPointer interface for all point record types
type LasPointer interface {
	Format() uint8
	PointData() *PointRecord0
	IsLateReturn() bool
	IsFirstReturn() bool
	IsIntermediateReturn() bool
	GpsTimeData() float64
	RgbData() *RgbData
}

// PointRecord0 is a LAS point record type 0.
type PointRecord0 struct {
	X             float64
	Y             float64
	Z             float64
	Intensity     uint16
	BitField      PointBitField
	ClassBitField ClassificationBitField
	ScanAngle     int8
	UserData      uint8
	PointSourceID uint16
}

// Format returns the point format number.
func (p *PointRecord0) Format() uint8 {
	return 0
}

// PointData returns the point data (PointRecord0) for the LAS point.
func (p *PointRecord0) PointData() *PointRecord0 {
	return p
}

// GpsTimeData returns the GPS time data for the LAS point.
func (p *PointRecord0) GpsTimeData() float64 {
	return NoData
}

// RgbData returns the RGB colour data for the LAS point.
func (p *PointRecord0) RgbData() *RgbData {
	return &RgbData{}
}

// IsLateReturn returns true if the point is a last return.
func (p *PointRecord0) IsLateReturn() bool {
	return p.BitField.ReturnNumber() == p.BitField.NumberOfReturns()
}

// IsFirstReturn returns true if the point is a first return.
func (p *PointRecord0) IsFirstReturn() bool {
	if p.BitField.ReturnNumber() == uint8(1) && p.BitField.NumberOfReturns() > uint8(1) {
		return true
	}
	return false
}

// IsIntermediateReturn returns true if the point is an intermediate return.
func (p *PointRecord0) IsIntermediateReturn() bool {
	rn := p.BitField.ReturnNumber()
	if rn > uint8(1) && rn < p.BitField.NumberOfReturns() {
		return true
	}
	return false
}

// PointRecord1 is a LAS point record type 1
type PointRecord1 struct {
	*PointRecord0
	GPSTime float64
}

// Format returns the point format number.
func (p *PointRecord1) Format() uint8 {
	return 1
}

// GpsTimeData returns the point data (PointRecord0) for the LAS point.
func (p *PointRecord1) GpsTimeData() float64 {
	return p.GPSTime
}

// RgbData returns the RGB colour data for the LAS point.
func (p *PointRecord1) RgbData() *RgbData {
	return &RgbData{}
}

// PointRecord2 is a LAS point record type 2
type PointRecord2 struct {
	*PointRecord0
	RGB *RgbData
}

// Format returns the point format number.
func (p *PointRecord2) Format() uint8 {
	return 2
}

// GpsTimeData returns the point data (PointRecord0) for the LAS point.
func (p *PointRecord2) GpsTimeData() float64 {
	return NoData
}

// RgbData returns the RGB colour data for the LAS point.
func (p *PointRecord2) RgbData() *RgbData {
	return p.RGB
}

// PointRecord3 is a LAS point record type 3
type PointRecord3 struct {
	*PointRecord0
	GPSTime float64
	RGB     *RgbData
}

// Format returns the point format number.
func (p *PointRecord3) Format() uint8 {
	return 3
}

// GpsTimeData returns the point data (PointRecord0) for the LAS point.
func (p *PointRecord3) GpsTimeData() float64 {
	return p.GPSTime
}

// RgbData returns the RGB colour data for the LAS point.
func (p *PointRecord3) RgbData() *RgbData {
	return p.RGB
}

// PointBitField is a point record bit field
type PointBitField struct {
	Value byte
}

// ReturnNumber returns the return number of the point
func (p *PointBitField) ReturnNumber() byte {
	ret := (p.Value & byte(7))
	if ret == 0 {
		ret = 1
	}
	return ret
}

// NumberOfReturns returns the number of returns of the point
func (p *PointBitField) NumberOfReturns() byte {
	ret := (p.Value & byte(56))
	if ret == 0 {
		ret = 1
	}
	return ret
}

// ScanDirectionFlag scan direction flag, `true` if moving from the left side of the
// in-track direction to the right side and false the opposite.
func (p *PointBitField) ScanDirectionFlag() bool {
	return (p.Value & byte(64)) == byte(64)
}

// EdgeOfFlightlineFlag Edge of flightline flag
func (p *PointBitField) EdgeOfFlightlineFlag() bool {
	return (p.Value & byte(128)) == byte(128)
}

// ClassificationBitField is a point record classification bit field
type ClassificationBitField struct {
	Value byte
}

// Classification of LAS point record
func (c *ClassificationBitField) Classification() byte {
	return c.Value & uint8(31)
}

// SetClassification sets the class value for a LAS point record
func (c *ClassificationBitField) SetClassification(value uint8) {
	c.Value = (c.Value & uint8(224)) | (value & uint8(31))
}

// ClassificationString returns a string represenation of the classiciation type.
func (c *ClassificationBitField) ClassificationString() string {
	classVal := c.Classification()
	switch {
	case classVal == 0:
		return "Created, never classified"
	case classVal == 1:
		return "Unclassified"
	case classVal == 2:
		return "Ground"
	case classVal == 3:
		return "Low vegetation"
	case classVal == 4:
		return "Medium vegetation"
	case classVal == 5:
		return "High vegetation"
	case classVal == 6:
		return "Building"
	case classVal == 7:
		return "Low point (noise)"
	case classVal == 8:
		return "Reserved"
	case classVal == 9:
		return "Water"
	case classVal == 10:
		return "Rail"
	case classVal == 11:
		return "Road surface"
	case classVal == 12:
		return "Reserved"
	case classVal == 13:
		return "Wire – guard (shield)"
	case classVal == 14:
		return "Wire – conductor (phase)"
	case classVal == 15:
		return "Transmission tower"
	case classVal == 16:
		return "Wire-structure connector (e.g. insulator)"
	case classVal == 17:
		return "Bridge deck"
	case classVal == 18:
		return "High Noise"
	case classVal >= 19 && classVal <= 63:
		return "Reserved"
	case classVal >= 64 && classVal <= 255:
		return "User defined"
	default:
		return "Unknown class"
	}
}

// Synthetic returns `true` if the point is synthetic, `false` otherwise
func (c *ClassificationBitField) Synthetic() bool {
	return (c.Value & uint8(32)) == uint8(32)
}

// SetSynthetic sets the value of synthetic for the point
func (c *ClassificationBitField) SetSynthetic(val bool) {
	if val {
		c.Value = c.Value | uint8(32)
	} else {
		c.Value = c.Value & uint8(223)
	}
}

// Keypoint returns `true` if the point is a keypoint, `false` otherwise
func (c *ClassificationBitField) Keypoint() bool {
	return (c.Value & uint8(64)) == uint8(64)
}

// SetKeypoint sets the value of the keypoint field for this point
func (c *ClassificationBitField) SetKeypoint(val bool) {
	if val {
		c.Value = c.Value | uint8(64)
	} else {
		c.Value = c.Value & uint8(191)
	}
}

// Withheld returns `true` if the point is withehld, `false` otherwise
func (c *ClassificationBitField) withheld() bool {
	return (c.Value & uint8(128)) == uint8(128)
}

// SetWithheld sets the value of the withheld field for this point
func (c *ClassificationBitField) SetWithheld(val bool) {
	if val {
		c.Value = c.Value | uint8(128)
	} else {
		c.Value = c.Value & uint8(127)
	}
}

// RgbData holds LAS point red-green-blue colour data
type RgbData struct {
	Red   uint16
	Green uint16
	Blue  uint16
}

// Creates a fixed-length string with buffer characters as null
func fixedLengthString(s string, length int) string {
	var b bytes.Buffer
	for n := 0; n < length; n++ {
		if n < len(s) && n < length {
			b.WriteString(string(s[n]))
		} else {
			b.WriteString("\x00")
		}
	}
	return b.String()
}
