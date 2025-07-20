package golas

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

type GPSTimeType uint8

const (
	GPSWeekTime             GPSTimeType = 0
	GPSAdjustedStandardTime GPSTimeType = 1
)

// Las allows to read LAS file format data. The supported LAS versions range from 1.1 to 1.4.
type Las struct {
	Header  LasHeader
	VLRs    []VLR
	EVLRs   []EVLR
	wkt     *WKT
	geotiff *GeoTIFFMetadata
	r       io.ReadSeeker
	current uint64
	sync.Mutex
}

// LasHeader models a LAS header. The structure is compatible with format versions 1.1 to 1.4.
// Fields absent in one version will be left at their default zero values.
type LasHeader struct {
	FileSignature                   string
	FileSourceId                    uint16
	GlobalEncoding                  GlobalEncoding
	ProjectIdGUID1                  uint32
	ProjectIdGUID2                  uint16
	ProjectIdGUID3                  uint16
	ProjectIdGUID4                  string
	VersionMajor                    uint8
	VersionMinor                    uint8
	SystemIdentifier                string
	GeneratingSoftware              string
	CreationDay                     uint16
	CreationYear                    uint16
	HeaderSize                      uint16
	OffsetToPointData               uint32
	NumberOfVLRs                    uint32
	PointDataRecordFormat           uint8
	PointDataRecordLength           uint16
	LegacyNumberOfPointRecords      uint32
	LegacyNumberOfPointByReturn     [5]uint32
	XScaleFactor                    float64
	YScaleFactor                    float64
	ZScaleFactor                    float64
	XOffset                         float64
	YOffset                         float64
	ZOffset                         float64
	MaxX                            float64
	MaxY                            float64
	MaxZ                            float64
	MinX                            float64
	MinY                            float64
	MinZ                            float64
	StartOfWaveformDataPacketRecord uint64
	StartOfFirstEVLR                uint64
	NumberOfEVLRs                   uint32
	NumberOfPointRecords            uint64
	NumberOfPointsByReturn          [15]uint64
}

// GlobalEncoding wraps the global header encoding flags of the LAS
type GlobalEncoding struct {
	GPSTime                GPSTimeType
	InternalWaveformData   bool
	ExternalWaveformData   bool
	SyntheticReturnNumbers bool
	WKT                    bool
}

// VLR contains the raw uninterpreted data of a VLR record.
type VLR struct {
	UserID                  string
	RecordID                uint16
	RecordLengthAfterHeader uint16
	Description             string
	Data                    []byte
}

// EVLR contains the raw uninterpreted data of an Extended VLR (EVLR) record.
type EVLR struct {
	UserID                  string
	RecordID                uint16
	RecordLengthAfterHeader uint64
	Description             string
	Data                    []byte
}

// WKT contains the WKT definitions of the LAS CoordinateSystem and MathTransform
// as extracted from VLR or EVLR records
type WKT struct {
	CoordinateSystem string
	MathTransform    string
}

// NewLas returns a new Las struct to read Points from a LAS datasource. It requires in input
// a io.ReadSeeker, which will be internally buffered. Returns an error in case of issues reading
// or parsing the LAS Header, its VLRs or its Extended VLRs.
func NewLas(r io.ReadSeeker) (*Las, error) {
	g := &Las{
		r: newBufferedReadSeeker(r),
	}
	err := g.readHeader()
	if err != nil {
		return nil, err
	}
	err = g.readVLRs()
	if err != nil {
		return nil, err
	}
	err = g.readEVLRs()
	if err != nil {
		return nil, err
	}
	g.extractWKTVLRs()
	err = g.extractGeoTIFF()
	if err != nil {
		return g, err
	}
	// prepare the reader to read point data
	_, err = g.r.Seek(int64(g.Header.OffsetToPointData), io.SeekStart)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// WKT returns the WKT object storing the LAS coordinate reference system metadata, if any was
// found in the LAS VLRs and EVLRs, else it returns nil.
func (g *Las) WKT() *WKT {
	return g.wkt
}

// GeoTIFFMetadata returns the GeoTIFFMetadata object storing the LAS GeoTIFFTag keys and values
// as parsed from the LAS VLRs and EVLRs, if any, else it returns nil.
func (g *Las) GeoTIFFMetadata() *GeoTIFFMetadata {
	return g.geotiff
}

// CRS returns either the EPSG code of the CRS in use extracted from the Geotiff metadata or returns a WKT string representing the Coordinate System embedded in the LAS.
// Emtpy string means no CRS metadata has been found, or it could not be extracted as string and requires
// advanced interpretation of the Geotiff metadata.
// If WKT Coordinate System information is available, it takes precedence over the Geotiff metadata.
// The EPSG code returned in case of GeoTIFF will have the form EPSG:XYZ (in case of geographic or projected CRS) or
// the form EPSG:XYZ+LMN in case the GeoTIFF also declares the presence of a Vertical CRS.
// Geotiff codes outside the EPSG ranges are skipped.
func (g *Las) CRS() string {
	if wkt := g.WKT(); wkt != nil {
		return wkt.CoordinateSystem
	}
	if geotiff := g.GeoTIFFMetadata(); geotiff != nil {
		// Vertical CS, if any
		// valid EPSG values should be between 5000 and 5999
		verticalCS := ""
		if val := geotiff.Keys[4096]; val != nil {
			code := val.AsShort()
			if code >= 5000 && code < 5999 {
				verticalCS = fmt.Sprintf("+%d", val.AsShort())
			}
		}
		// ProjectedCSType is key 3072
		// valid EPSG values should be between 20000 and 32760
		if val := geotiff.Keys[3072]; val != nil {
			code := val.AsShort()
			if code >= 20000 && code < 32760 {
				return fmt.Sprintf("EPSG:%d%s", val.AsShort(), verticalCS)
			}
		}
		// GeographicTypeGeoKey is key 2048
		// valid EPSG values should be between 4000 and 4999
		if val := geotiff.Keys[2048]; val != nil {
			code := val.AsShort()
			if code >= 4000 && code < 5000 {
				return fmt.Sprintf("EPSG:%d%s", code, verticalCS)
			}
		}
	}
	return ""
}

// NumberOfPoints returns the number of points in the LAS. This method
// interprets appropriately the LAS header fields based on the file format version
// and is reccomended to be used over reading the raw header values.
func (g *Las) NumberOfPoints() uint64 {
	if g.Header.VersionMinor < 4 {
		return uint64(g.Header.LegacyNumberOfPointRecords)
	}
	return g.Header.NumberOfPointRecords
}

// HasNext is a heuristic to understand whether there are more points to be read from the LAS file
func (g *Las) HasNext() bool {
	g.Lock()
	defer g.Unlock()
	return g.current < g.NumberOfPoints()
}

// Next returns the next unread point from the LAS, or an error in case of read or data parsing errors.
// An error should be considered as an unrecoverable event and no further invocations of the Next function should be issued.
func (g *Las) Next() (Point, error) {
	p := Point{
		PointDataRecordFormat: g.Header.PointDataRecordFormat,
	}

	// Read raw point data
	// The reader is already at the right position as the NewLas method performs
	// a Seek operation right before returning the struct
	g.Lock()
	if g.current >= g.NumberOfPoints() {
		g.Unlock()
		return p, io.EOF
	}
	data := make([]byte, g.Header.PointDataRecordLength)
	if _, err := io.ReadFull(g.r, data); err != nil {
		return p, io.ErrUnexpectedEOF
	}
	g.current++
	g.Unlock()

	// Interpret raw point data
	r := bytes.NewReader(data)

	// Read X and compute the real coordinates
	x, err := readLong(r)
	if err != nil {
		return p, err
	}
	p.X = (float64(x) * g.Header.XScaleFactor) + g.Header.XOffset

	// Read Y and compute the real coordinates
	y, err := readLong(r)
	if err != nil {
		return p, err
	}
	p.Y = (float64(y) * g.Header.YScaleFactor) + g.Header.YOffset

	// Read Z and compute the real coordinates
	z, err := readLong(r)
	if err != nil {
		return p, err
	}
	p.Z = (float64(z) * g.Header.ZScaleFactor) + g.Header.ZOffset

	// Read Intensity
	p.Intensity, err = readUnsignedShort(r)
	if err != nil {
		return p, err
	}

	// Read Flags byte 1
	p.flags1, err = r.ReadByte()
	if err != nil {
		return p, err
	}

	// Read Flags byte 2 for point formats >= 6
	if formatHasExtraFlagByte(p.PointDataRecordFormat) {
		p.flags2, err = r.ReadByte()
		if err != nil {
			return p, err
		}
	}

	// Read Classification Byte
	classRaw, err := r.ReadByte()
	if err != nil {
		return p, err
	}
	p.classificationRaw = classRaw

	// Parse Classification
	classification := classRaw
	if p.PointDataRecordFormat < 6 {
		// for point formats < 6 only keep the first 5 bits
		classification &= 0b00011111
	}
	p.Classification = classification

	// Read Scan Angle Rank for point formats supporting it
	if formatHasScanAngleRank(p.PointDataRecordFormat) {
		p.ScanAngleRank, err = readInt8(r)
		if err != nil {
			return p, err
		}
	}

	// Read User Data
	p.UserData, err = readUint8(r)
	if err != nil {
		return p, err
	}

	// Read Scan Angle for point formats supporting it
	if formatHasScanAngle(p.PointDataRecordFormat) {
		p.ScanAngle, err = readShort(r)
		if err != nil {
			return p, err
		}
	}

	// Read Point Source ID
	p.PointSourceID, err = readUnsignedShort(r)
	if err != nil {
		return p, err
	}

	// Read GPS Time for point formats supporting it
	if formatHasGpsTime(p.PointDataRecordFormat) {
		p.GPSTime, err = readFloat64(r)
		if err != nil {
			return p, err
		}
	}

	// Read Colors for point formats supporting them. Colors are kept as 16 bit of color depth.
	if formatHasRgbColors(p.PointDataRecordFormat) {
		p.Red, err = readUnsignedShort(r)
		if err != nil {
			return p, err
		}
		p.Green, err = readUnsignedShort(r)
		if err != nil {
			return p, err
		}
		p.Blue, err = readUnsignedShort(r)
		if err != nil {
			return p, err
		}
	}

	// Read NIR data for point formats supporting it.
	if formatHasNir(p.PointDataRecordFormat) {
		p.NIR, err = readUnsignedShort(r)
		if err != nil {
			return p, err
		}
	}

	// Read WavePackets metadata for point formats supporting it.
	if formatHasWavePackets(p.PointDataRecordFormat) {
		p.WavePacketDescriptorIndex, err = readUint8(r)
		if err != nil {
			return p, err
		}
		p.ByteOffsetToWaveformData, err = readUnsignedLong64(r)
		if err != nil {
			return p, err
		}
		p.WaveformPacketSizeBytes, err = readUnsignedLong(r)
		if err != nil {
			return p, err
		}
		p.ReturnPointWaveformLocation, err = readFloat32(r)
		if err != nil {
			return p, err
		}
		p.ParametricDx, err = readFloat32(r)
		if err != nil {
			return p, err
		}
		p.ParametricDy, err = readFloat32(r)
		if err != nil {
			return p, err
		}
		p.ParametricDz, err = readFloat32(r)
		if err != nil {
			return p, err
		}
	}

	// Read point custom data, all the residual bytes
	_, err = io.Copy(io.Discard, r)
	if err != nil {
		return p, err
	}

	// Return the point
	return p, nil
}

// readHeader reads the LAS header metadata from the las reader and sets it in the struct
func (g *Las) readHeader() error {
	g.Lock()
	defer g.Unlock()
	header := &LasHeader{}
	if err := g.readVersionAgnosticHeaderSection(header); err != nil {
		return err
	}
	if err := g.readVersionSpecificHeaderSection(header); err != nil {
		return err
	}
	g.Header = *header
	return nil
}

// readHeader reads the LAS header metadata that is common across all versions from 1.1 to 1.4
func (g *Las) readVersionAgnosticHeaderSection(header *LasHeader) error {
	var err error
	// Read and validate LASF signature
	if header.FileSignature, err = g.readString(4); err != nil || header.FileSignature != "LASF" {
		if err != nil {
			return err
		}
		return errors.New("unexpected file signature")
	}
	if header.FileSourceId, err = g.readUnsignedShort(); err != nil {
		return err
	}
	var geData uint16
	if geData, err = g.readUnsignedShort(); err != nil {
		return err
	}
	header.GlobalEncoding, err = globalEncodingFromUint16(geData)
	if err != nil {
		return err
	}
	if header.ProjectIdGUID1, err = g.readUnsignedLong(); err != nil {
		return err
	}
	if header.ProjectIdGUID2, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.ProjectIdGUID3, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.ProjectIdGUID4, err = g.readString(8); err != nil {
		return err
	}
	if header.VersionMajor, err = g.readUint8(); err != nil {
		return err
	}
	if header.VersionMinor, err = g.readUint8(); err != nil {
		return err
	}
	if header.VersionMajor != 1 {
		return fmt.Errorf("unsupported Version Major %d", header.VersionMajor)
	}
	if header.VersionMinor < 1 || header.VersionMinor > 4 {
		return fmt.Errorf("unsupported Version Minor %v", header.VersionMinor)
	}
	if header.SystemIdentifier, err = g.readString(32); err != nil {
		return err
	}
	if header.GeneratingSoftware, err = g.readString(32); err != nil {
		return err
	}
	if header.CreationDay, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.CreationYear, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.HeaderSize, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.OffsetToPointData, err = g.readUnsignedLong(); err != nil {
		return err
	}
	if header.NumberOfVLRs, err = g.readUnsignedLong(); err != nil {
		return err
	}
	if header.PointDataRecordFormat, err = g.readUint8(); err != nil {
		return err
	}
	if header.PointDataRecordLength, err = g.readUnsignedShort(); err != nil {
		return err
	}
	if header.LegacyNumberOfPointRecords, err = g.readUnsignedLong(); err != nil {
		return err
	}
	var data []uint32
	if data, err = g.readUnsignedLongArray(5); err != nil {
		return err
	}
	header.LegacyNumberOfPointByReturn = [5]uint32(data)
	if header.XScaleFactor, err = g.readFloat64(); err != nil {
		return err
	}
	if header.YScaleFactor, err = g.readFloat64(); err != nil {
		return err
	}
	if header.ZScaleFactor, err = g.readFloat64(); err != nil {
		return err
	}
	if header.XOffset, err = g.readFloat64(); err != nil {
		return err
	}
	if header.YOffset, err = g.readFloat64(); err != nil {
		return err
	}
	if header.ZOffset, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MaxX, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MinX, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MaxY, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MinY, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MaxZ, err = g.readFloat64(); err != nil {
		return err
	}
	if header.MinZ, err = g.readFloat64(); err != nil {
		return err
	}
	return nil
}

func (g *Las) readVersionSpecificHeaderSection(header *LasHeader) error {
	if header.VersionMinor == 3 {
		return g.readLas13SpecificHeaderBlock(header)
	}
	if header.VersionMinor == 4 {
		return g.readLas14SpecificHeaderBlock(header)
	}
	return nil
}

func (g *Las) readLas13SpecificHeaderBlock(header *LasHeader) error {
	val, err := g.readUnsignedLong64()
	if err != nil {
		return err
	}
	header.StartOfWaveformDataPacketRecord = val
	return nil
}

func (g *Las) readLas14SpecificHeaderBlock(header *LasHeader) error {
	var err error
	if header.StartOfWaveformDataPacketRecord, err = g.readUnsignedLong64(); err != nil {
		return err
	}
	if header.StartOfFirstEVLR, err = g.readUnsignedLong64(); err != nil {
		return err
	}
	if header.NumberOfEVLRs, err = g.readUnsignedLong(); err != nil {
		return err
	}
	if header.NumberOfPointRecords, err = g.readUnsignedLong64(); err != nil {
		return err
	}
	var temp []uint64
	if temp, err = g.readUnsignedLong64Array(15); err != nil {
		return err
	}
	header.NumberOfPointsByReturn = [15]uint64(temp)
	return nil
}

func (g *Las) readVLRs() error {
	g.Lock()
	defer g.Unlock()
	for i := 0; i < int(g.Header.NumberOfVLRs); i++ {
		v := VLR{}
		// reserved short, read and skip
		_, err := g.readUnsignedShort()
		if err != nil {
			return err
		}
		if v.UserID, err = g.readString(16); err != nil {
			return err
		}
		if v.RecordID, err = g.readUnsignedShort(); err != nil {
			return err
		}
		if v.RecordLengthAfterHeader, err = g.readUnsignedShort(); err != nil {
			return err
		}
		if v.Description, err = g.readString(32); err != nil {
			return err
		}
		data, err := g.readBytes(int(v.RecordLengthAfterHeader))
		if err != nil {
			return err
		}
		v.Data = data
		g.VLRs = append(g.VLRs, v)
	}
	return nil
}

func (g *Las) readEVLRs() error {
	g.Lock()
	defer g.Unlock()
	if g.Header.VersionMinor < 4 {
		// EVLRs are not really supported in LAS < 1.4
		return nil
	}
	g.r.Seek(int64(g.Header.StartOfFirstEVLR), io.SeekStart)
	for i := 0; i < int(g.Header.NumberOfEVLRs); i++ {
		v := EVLR{}
		res, err := g.readUnsignedShort()
		if err != nil {
			return err
		}
		// reserved should always be zero
		if res != 0 {
			return errors.New("found invalid EVLR with reserved header field not zero")
		}
		if v.UserID, err = g.readString(16); err != nil {
			return err
		}
		if v.RecordID, err = g.readUnsignedShort(); err != nil {
			return err
		}
		if v.RecordLengthAfterHeader, err = g.readUnsignedLong64(); err != nil {
			return err
		}
		if v.Description, err = g.readString(32); err != nil {
			return err
		}
		data, err := g.readBytes(int(v.RecordLengthAfterHeader))
		if err != nil {
			return err
		}
		v.Data = data
		g.EVLRs = append(g.EVLRs, v)
	}
	return nil
}

func (g *Las) extractGeoTIFF() error {
	geo := &GeoTIFFMetadata{
		Keys: map[int]*GeoTIFFKey{},
	}
	var geoDirectoryTagRaw, geoDoubleParamsTagRaw, geoAsciiParamsTagRaw []byte
	for _, v := range g.VLRs {
		if v.UserID == "LASF_Projection" && v.RecordID == 34735 {
			geoDirectoryTagRaw = v.Data
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 34736 {
			geoDoubleParamsTagRaw = v.Data
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 34737 {
			geoAsciiParamsTagRaw = v.Data
		}
	}
	for _, v := range g.EVLRs {
		if v.UserID == "LASF_Projection" && v.RecordID == 34735 {
			geoDirectoryTagRaw = v.Data
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 34736 {
			geoDoubleParamsTagRaw = v.Data
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 34737 {
			geoAsciiParamsTagRaw = v.Data
		}
	}
	if len(geoDirectoryTagRaw) == 0 {
		// no geo directory tag raw found, return
		return nil
	}
	r := bytes.NewReader(geoDirectoryTagRaw)
	readUint16 := func(r *bytes.Reader) (uint16, error) {
		var val uint16
		err := binary.Read(r, binary.LittleEndian, &val)
		return val, err
	}
	readFloat64 := func(b []byte, offset int) (float64, error) {
		if offset < 0 || offset+8 > len(b) {
			return 0, errors.New("offset out of bounds")
		}
		var v float64
		err := binary.Read(bytes.NewReader(b[offset:offset+8]), binary.LittleEndian, &v)
		return v, err
	}
	wKeyDirectoryVersion, err := readUint16(r)
	if err != nil {
		return err
	}
	if wKeyDirectoryVersion != 1 {
		return errors.New("wKeyDirectoryVersion should be 1")
	}
	wKeyRevision, err := readUint16(r)
	if err != nil {
		return err
	}
	if wKeyRevision != 1 {
		return errors.New("wKeyRevision should be 1")
	}
	wMinorRevision, err := readUint16(r)
	if err != nil {
		return err
	}
	if wMinorRevision != 0 {
		return errors.New("wMinorRevision should be 0")
	}
	wNumberOfKeys, err := readUint16(r)
	if err != nil {
		return err
	}
	for i := 0; i < int(wNumberOfKeys); i++ {
		wKeyID, err := readUint16(r)
		if err != nil {
			return err
		}
		wTIFFTagLocation, err := readUint16(r)
		if err != nil {
			return err
		}
		wCount, err := readUint16(r)
		if err != nil {
			return err
		}
		wValueOffset, err := readUint16(r)
		if err != nil {
			return err
		}
		if wTIFFTagLocation == 0 {
			geo.Keys[int(wKeyID)] = &GeoTIFFKey{
				KeyId:    int(wKeyID),
				Type:     GTTagTypeShort,
				RawValue: wValueOffset,
			}
		} else if wTIFFTagLocation == 34736 {
			value, err := readFloat64(geoDoubleParamsTagRaw, int(wValueOffset)*8)
			if err != nil {
				return err
			}
			geo.Keys[int(wKeyID)] = &GeoTIFFKey{
				KeyId:    int(wKeyID),
				Type:     GTTagTypeDouble,
				RawValue: value,
			}
		} else if wTIFFTagLocation == 34737 {
			if int(wValueOffset+wCount) > len(geoAsciiParamsTagRaw) {
				return errors.New("wValueOffset out of bounds")
			}
			value := strings.TrimRight(string(geoAsciiParamsTagRaw[int(wValueOffset):int(wValueOffset)+int(wCount)]), "\u0000|")
			if err != nil {
				return err
			}
			geo.Keys[int(wKeyID)] = &GeoTIFFKey{
				KeyId:    int(wKeyID),
				Type:     GTTagTypeString,
				RawValue: value,
			}
		}
	}
	g.geotiff = geo
	return nil
}

func (g *Las) extractWKTVLRs() {
	w := &WKT{}
	for _, v := range g.VLRs {
		if v.UserID == "LASF_Projection" && v.RecordID == 2111 {
			w.MathTransform = strings.TrimRight(string(v.Data), "\u0000")
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 2112 {
			w.CoordinateSystem = strings.TrimRight(string(v.Data), "\u0000")
		}
	}
	for _, v := range g.EVLRs {
		if v.UserID == "LASF_Projection" && v.RecordID == 2111 {
			w.MathTransform = strings.TrimRight(string(v.Data), "\u0000")
		}
		if v.UserID == "LASF_Projection" && v.RecordID == 2112 {
			w.CoordinateSystem = strings.TrimRight(string(v.Data), "\u0000")
		}
	}
	if w.CoordinateSystem != "" || w.MathTransform != "" {
		g.wkt = w
	}
}

func (g *Las) readString(n int) (string, error) {
	return readString(g.r, n)
}

func (g *Las) readUint8() (uint8, error) {
	return readUint8(g.r)
}

func (g *Las) readUnsignedShort() (uint16, error) {
	return readUnsignedShort(g.r)
}

func (g *Las) readUnsignedLong() (uint32, error) {
	return readUnsignedLong(g.r)
}

func (g *Las) readUnsignedLong64() (uint64, error) {
	return readUnsignedLong64(g.r)
}

func (g *Las) readFloat64() (float64, error) {
	return readFloat64(g.r)
}

func (g *Las) readUnsignedLongArray(n int) ([]uint32, error) {
	return readUnsignedLongArray(g.r, n)
}

func (g *Las) readUnsignedLong64Array(n int) ([]uint64, error) {
	return readUnsignedLong64Array(g.r, n)
}

func (g *Las) readBytes(n int) ([]byte, error) {
	return readBytes(g.r, n)
}

func globalEncodingFromUint16(b uint16) (GlobalEncoding, error) {
	g := GlobalEncoding{}
	g.GPSTime = GPSTimeType(b & 0b1)
	g.InternalWaveformData = (b & 0b10 >> 1) == 1
	g.ExternalWaveformData = (b & 0b100 >> 2) == 1
	if g.InternalWaveformData && g.ExternalWaveformData {
		return g, errors.New("internal and external waveform data bits cannot be both set")
	}
	g.SyntheticReturnNumbers = (b & 0b1000 >> 3) == 1
	g.WKT = (b & 0b10000 >> 4) == 1
	return g, nil
}
