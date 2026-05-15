package golas

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

func compareStructs(a, b interface{}, tolerance float64) (bool, error) {
	// Ensure both inputs are structs
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)

	if valA.Kind() != reflect.Struct || valB.Kind() != reflect.Struct {
		return false, fmt.Errorf("both inputs must be structs")
	}

	// Check that the structs have the same type
	if valA.Type() != valB.Type() {
		return false, fmt.Errorf("structs must be of the same type")
	}

	// Iterate through the fields of the struct
	for i := 0; i < valA.NumField(); i++ {
		fieldA := valA.Field(i)
		fieldB := valB.Field(i)

		// Skip unexported fields
		if !fieldA.CanInterface() || !fieldB.CanInterface() {
			continue
		}

		// Handle floating-point fields with tolerance
		if fieldA.Kind() == reflect.Float32 || fieldA.Kind() == reflect.Float64 {
			diff := math.Abs(fieldA.Float() - fieldB.Float())
			if diff > tolerance {
				return false, nil
			}
		} else if fieldA.Kind() == reflect.Slice && fieldA.Type().Elem().Kind() == reflect.Uint8 {
			// Handle []byte fields
			if !bytes.Equal(fieldA.Bytes(), fieldB.Bytes()) {
				return false, nil
			}
		} else if !reflect.DeepEqual(fieldA.Interface(), fieldB.Interface()) {
			// Use DeepEqual for other fields
			return false, nil
		}
	}

	return true, nil
}

func TestLasPoints(t *testing.T) {
	files := map[string]Point{
		"1.1_0.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 0,
			Classification:        2,
			ScanAngleRank:         -13,
		},
		"1.1_1.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 1,
			Classification:        2,
			ScanAngleRank:         -13,
			GPSTime:               1205902800.0,
		},
		"1.2_0.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 0,
			Classification:        2,
			ScanAngleRank:         -13,
		},
		"1.2_1.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 1,
			Classification:        2,
			ScanAngleRank:         -13,
			GPSTime:               1205902800.0,
		},
		"1.2_2.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 2,
			Classification:        2,
			ScanAngleRank:         -13,
			Red:                   255,
			Green:                 12,
			Blue:                  234,
		},
		"1.2_3.las": {
			X:                     470692.440000,
			Y:                     4602888.900000,
			Z:                     16.000000,
			PointDataRecordFormat: 3,
			Classification:        2,
			ScanAngleRank:         -13,
			GPSTime:               1205902800.0,
			Red:                   255,
			Green:                 12,
			Blue:                  234,
		},
	}
	for filename, pt := range files {
		f, err := os.Open(path.Join("./testdata", filename))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		l, err := NewLas(f)
		if err != nil {
			t.Fatal(err)
		}
		if actual := l.NumberOfPoints(); actual != 1 {
			t.Fatalf("unexpected number of points, expected 1 got %d", actual)
		}
		actual, err := l.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v, %v", err, l.Header)
		}
		if eq, err := compareStructs(pt, actual, 0); err != nil || !eq {
			t.Errorf("expected point %v, got %v", pt, actual)
		}
		if actual := l.CRS(); actual != "EPSG:26915" {
			t.Errorf("expected CRS %s, got %s", "EPSG:26915", actual)
		}
		if actual := actual.ReturnNumber(); actual != 2 {
			t.Errorf("expected return number %d, got %d", 2, actual)
		}
		if actual := actual.NumberOfReturns(); actual != 0 {
			t.Errorf("expected number of returns %d, got %d", 0, actual)
		}
		if actual := len(actual.ClassificationFlags()); actual != 0 {
			t.Errorf("expected no classification flags, got %d", actual)
		}
		if actual := actual.EdgeOfFlightLineFlag(); actual != EOFNormal {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", EOFNormal, actual)
		}
		if actual := actual.ScanDirectionFlag(); actual != ScanDirectionNegative {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", ScanDirectionNegative, actual)
		}
		if actual := actual.ScannerChannel(); actual != 0 {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", 0, actual)
		}
	}
}

func TestLasGeometries(t *testing.T) {
	entries, err := os.ReadDir("./testdata")
	if err != nil {
		t.Fatal(err)
	}

	basic := []Point{
		{X: 432488.4714159001, Y: 4.705678720925195e+06, Z: 2.550538045727727, Red: 160, Green: 166, Blue: 203, Intensity: 7, Classification: 3},
		{X: 432466.58372129063, Y: 4.705686414284739e+06, Z: 4.457767479175496, Red: 186, Green: 200, Blue: 237, Intensity: 7, Classification: 3},
		{X: 432469.98167019564, Y: 4.705681849831438e+06, Z: 4.655027394763156, Red: 156, Green: 167, Blue: 204, Intensity: 7, Classification: 3},
		{X: 432456.1696575739, Y: 4.705683863015449e+06, Z: 1.7922372985151949, Red: 107, Green: 114, Blue: 156, Intensity: 7, Classification: 3},
		{X: 432465.34641605255, Y: 4.705682466042181e+06, Z: 1.9543476118949155, Red: 165, Green: 176, Blue: 206, Intensity: 7, Classification: 3},
		{X: 432471.7795208326, Y: 4.705664509499522e+06, Z: 1.9162578617315837, Red: 90, Green: 136, Blue: 213, Intensity: 7, Classification: 3},
		{X: 432449.48655283387, Y: 4.705672817604579e+06, Z: 2.7424278273838762, Red: 51, Green: 66, Blue: 87, Intensity: 7, Classification: 3},
		{X: 432457.86106384156, Y: 4.705682649635591e+06, Z: 2.081788046497537, Red: 80, Green: 97, Blue: 123, Intensity: 7, Classification: 3},
		{X: 432466.3000286405, Y: 4.705680013894157e+06, Z: 1.9357872046388636, Red: 160, Green: 169, Blue: 202, Intensity: 7, Classification: 3},
		{X: 432455.7336230836, Y: 4.705657784890012e+06, Z: 6.120847715554042, Red: 73, Green: 78, Blue: 110, Intensity: 7, Classification: 3},
	}
	var expectedNumPoints uint64 = 10
	for _, e := range entries {
		filename := e.Name()
		if filename[0:5] != "las-1" {
			continue
		}
		f, err := os.Open(path.Join("./testdata", filename))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		l, err := NewLas(f)
		if err != nil {
			t.Fatalf("unexpected error opening %v", err)
		}
		if actual := l.NumberOfPoints(); actual != uint64(expectedNumPoints) {
			t.Errorf("unexpected number of points for %s: expected %d got %d", filename, expectedNumPoints, actual)
		}
		r, err := regexp.Compile(`las-(\d)(\d)-pf(\d)(-sf)?`)
		if err != nil {
			t.Fatalf("unexpected regex error: %v", err)
		}
		for i := 0; i < 10; i++ {
			actual, err := l.Next()
			if err != nil {
				t.Errorf("%s: unexpected error returned: %v", filename, err)
			}
			if actual := actual.ReturnNumber(); actual != 0 {
				t.Errorf("%s: expected return number %d, got %d", filename, 0, actual)
			}
			if actual := actual.NumberOfReturns(); actual != 0 {
				t.Errorf("%s: expected number of returns %d, got %d", filename, 0, actual)
			}
			if actual := actual.EdgeOfFlightLineFlag(); actual != EOFNormal {
				t.Errorf("%s: expected EdgeOfFlightLineFlag %d, got %d", filename, EOFNormal, actual)
			}
			if actual := actual.ScanDirectionFlag(); actual != ScanDirectionNegative {
				t.Errorf("%s: expected EdgeOfFlightLineFlag %d, got %d", filename, ScanDirectionNegative, actual)
			}
			if actual := actual.ScannerChannel(); actual != 0 {
				t.Errorf("%s: expected EdgeOfFlightLineFlag %d, got %d", filename, 0, actual)
			}
			matches := r.FindStringSubmatch(filename)
			if matches == nil {
				t.Fatalf("%s: unexpected regex error", filename)
			}
			pf, err := strconv.Atoi(matches[3])
			if err != nil {
				t.Fatal(err)
			}
			expected := basic[i]
			expected.PointDataRecordFormat = uint8(pf)
			if !formatHasRgbColors(expected.PointDataRecordFormat) {
				expected.Red = 0
				expected.Green = 0
				expected.Blue = 0
			} else {
				expected.Red *= 256
				expected.Green *= 256
				expected.Blue *= 256
			}
			if expected.PointDataRecordFormat == 7 {
				expected.Classification = 90
			}
			if ok, err := compareStructs(expected, actual, 0.000001); !ok || err != nil {
				t.Errorf("for file %s, expected point %v got %v", filename, expected, actual)
			}
			if pf != 7 {
				if actual := len(actual.ClassificationFlags()); actual != 1 {
					t.Fatalf("%s: expected 1 classification flags, got %d", filename, actual)
				}
				if actual := actual.ClassificationFlags()[0]; actual != ClassificationSynthetic {
					t.Errorf("%s: expected flag %v, got %v", filename, ClassificationSynthetic, actual)
				}
			} else {
				if actual := len(actual.ClassificationFlags()); actual != 0 {
					t.Fatalf("%s: expected no classification flags, got %d", filename, actual)
				}
			}
		}
	}
}

func TestLasSimple(t *testing.T) {
	f, err := os.Open("./testdata/simple.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 1065
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		_, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		GeneratingSoftware:         "TerraScan",
		HeaderSize:                 227,
		OffsetToPointData:          227,
		PointDataRecordFormat:      3,
		PointDataRecordLength:      34,
		LegacyNumberOfPointRecords: 1065,
		LegacyNumberOfPointByReturn: [5]uint32{
			925, 114, 21, 5, 0,
		},
		XScaleFactor: 0.01,
		YScaleFactor: 0.01,
		ZScaleFactor: 0.01,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         638982.55,
		MaxY:         853535.43,
		MaxZ:         586.38,
		MinX:         635619.85,
		MinY:         848899.7000000001,
		MinZ:         406.59000000000003,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
}

func TestLas12EmptyGeotiff(t *testing.T) {
	f, err := os.Open("./testdata/1.2-empty-geotiff-vlrs.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 43
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		actual, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		actualCount++
		if actualCount < 42 && actualCount != 28 {
			if actual := actual.ReturnNumber(); actual != 1 {
				t.Errorf("expected return number %d, got %d", 1, actual)
			}
			if actual := actual.NumberOfReturns(); actual != 1 {
				t.Errorf("%d,expected number of returns %d, got %d", actualCount, 1, actual)
			}
		} else if actualCount >= 42 {
			if actual := actual.ReturnNumber(); actual != 2 {
				t.Errorf("expected return number %d, got %d", 2, actual)
			}
			if actual := actual.NumberOfReturns(); actual != 2 {
				t.Errorf("%d,expected number of returns %d, got %d", actualCount, 2, actual)
			}
		} else if actualCount == 28 {
			if actual := actual.ReturnNumber(); actual != 1 {
				t.Errorf("expected return number %d, got %d", 1, actual)
			}
			if actual := actual.NumberOfReturns(); actual != 2 {
				t.Errorf("%d,expected number of returns %d, got %d", actualCount, 2, actual)
			}
		}
		if actual := len(actual.ClassificationFlags()); actual != 0 {
			t.Errorf("expected no classification flags, got %d", actual)
		}
		if actual := actual.EdgeOfFlightLineFlag(); actual != EOFNormal {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", EOFNormal, actual)
		}
		if actual := actual.ScanDirectionFlag(); actual != ScanDirectionNegative {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", ScanDirectionNegative, actual)
		}
		if actual := actual.ScannerChannel(); actual != 0 {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", 0, actual)
		}
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		SystemIdentifier:           "EXTRACTION",
		GeneratingSoftware:         "rdbconvert",
		CreationDay:                242,
		CreationYear:               2016,
		HeaderSize:                 227,
		OffsetToPointData:          8398,
		NumberOfVLRs:               5,
		PointDataRecordFormat:      1,
		PointDataRecordLength:      34,
		LegacyNumberOfPointRecords: 43,
		LegacyNumberOfPointByReturn: [5]uint32{
			41, 2, 0, 0, 0,
		},
		XScaleFactor: 0.00025,
		YScaleFactor: 0.00025,
		ZScaleFactor: 0.00025,
		XOffset:      34.81025,
		YOffset:      -28.986,
		ZOffset:      60.2725,
		MaxX:         211.08525,
		MaxY:         81.46075,
		MaxZ:         3.283250000000002,
		MinX:         -25.79175,
		MinY:         -15.9695,
		MinZ:         -13.1125,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
}

func TestLas12WithColor(t *testing.T) {
	f, err := os.Open("./testdata/1.2-with-color.las")
	if err != nil {
		t.Fatal(err)
	}
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 1065
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		p, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		if p.Red+p.Green+p.Blue == 0 {
			t.Errorf("expected point to be colored: %d, %d, %d", p.Red, p.Green, p.Blue)
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		GeneratingSoftware:         "TerraScan",
		HeaderSize:                 227,
		OffsetToPointData:          229,
		PointDataRecordFormat:      3,
		PointDataRecordLength:      34,
		LegacyNumberOfPointRecords: 1065,
		LegacyNumberOfPointByReturn: [5]uint32{
			925, 114, 21, 5, 0,
		},
		XScaleFactor: 0.01,
		YScaleFactor: 0.01,
		ZScaleFactor: 0.01,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         638982.55,
		MaxY:         853535.43,
		MaxZ:         586.38,
		MinX:         635619.85,
		MinY:         848899.7000000001,
		MinZ:         406.59000000000003,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
}

func TestLas12WithColorClipped(t *testing.T) {
	f, err := os.Open("./testdata/1.2-with-color-clipped.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedDeclaredNumberOfPoints := 1065
	expectedCount := 1064
	if actual := l.NumberOfPoints(); actual != uint64(expectedDeclaredNumberOfPoints) {
		t.Errorf("expected %d points, got %d", expectedDeclaredNumberOfPoints, actual)
	}
	actualCount := 0

	for l.HasNext() {
		_, err = l.Next()
		if err != nil {
			break
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}
	if err == nil {
		t.Fatalf("expected read error but got none")
	}
	if err == io.EOF {
		t.Errorf("io.EOF returned but expected different error")
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		GeneratingSoftware:         "TerraScan",
		HeaderSize:                 227,
		OffsetToPointData:          229,
		PointDataRecordFormat:      3,
		PointDataRecordLength:      34,
		LegacyNumberOfPointRecords: 1065,
		LegacyNumberOfPointByReturn: [5]uint32{
			925, 114, 21, 5, 0,
		},
		XScaleFactor: 0.01,
		YScaleFactor: 0.01,
		ZScaleFactor: 0.01,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         638982.55,
		MaxY:         853535.43,
		MaxZ:         586.38,
		MinX:         635619.85,
		MinY:         848899.7000000001,
		MinZ:         406.59000000000003,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
}

func TestLasBadVlrCount(t *testing.T) {
	f, err := os.Open("./testdata/bad_vlr_count.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err == nil {
		t.Fatal("expected error but got none")
	}
	if l != nil {
		t.Fatalf("expected nil golas but got %v", l)
	}
}

func TestLasEPSG4326(t *testing.T) {
	f, err := os.Open("./testdata/epsg_4326.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 5380
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		_, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		SystemIdentifier:           "MODIFICATION",
		GeneratingSoftware:         "QT Modeler",
		HeaderSize:                 227,
		OffsetToPointData:          853,
		NumberOfVLRs:               3,
		PointDataRecordFormat:      0,
		PointDataRecordLength:      20,
		LegacyNumberOfPointRecords: 5380,
		LegacyNumberOfPointByReturn: [5]uint32{
			5380, 0, 0, 0, 0,
		},
		XScaleFactor: 1e-7,
		YScaleFactor: 1e-7,
		ZScaleFactor: 1e-7,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         -94.66063109999999,
		MaxY:         31.0473291,
		MaxZ:         78.1190002,
		MinX:         -94.68346539999999,
		MinY:         31.0367341,
		MinZ:         39.0810002,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
	if len(l.VLRs) != 3 {
		t.Errorf("expected 3 VLRs, got %d", len(l.VLRs))
	}
	v := l.VLRs[0]
	if v.RecordID != 34735 {
		t.Errorf("expected VLR with record id %d, got %d", 34735, v.RecordID)
	}
	if v.UserID != "LASF_Projection" {
		t.Errorf("expected VLR with user id %s, got %s", "LASF_Projection", v.UserID)
	}
	if v.RecordLengthAfterHeader != 64 {
		t.Errorf("expected VLR with length %d, got %d", 64, v.RecordLengthAfterHeader)
	}
	v = l.VLRs[1]
	if v.RecordID != 34736 {
		t.Errorf("expected VLR with record id %d, got %d", 34736, v.RecordID)
	}
	if v.UserID != "LASF_Projection" {
		t.Errorf("expected VLR with user id %s, got %s", "LASF_Projection", v.UserID)
	}
	if v.RecordLengthAfterHeader != 16 {
		t.Errorf("expected VLR with length %d, got %d", 16, v.RecordLengthAfterHeader)
	}
	v = l.VLRs[2]
	if v.RecordID != 34737 {
		t.Errorf("expected VLR with record id %d, got %d", 34737, v.RecordID)
	}
	if v.UserID != "LASF_Projection" {
		t.Errorf("expected VLR with user id %s, got %s", "LASF_Projection", v.UserID)
	}
	if v.RecordLengthAfterHeader != 7 {
		t.Errorf("expected VLR with length %d, got %d", 7, v.RecordLengthAfterHeader)
	}
	if actual := l.CRS(); actual != "EPSG:4326" {
		t.Errorf("expected crs %s, got %v", "EPSG:4326", actual)
	}
	expectedKeyValues := map[int]any{
		1024: 2,
		1025: 1,
		2048: 4326,
		2049: "WGS 84",
		2054: 9102,
		2057: 6378137.0,
		2059: 298.257223563,
	}
	for key, val := range l.GeoTIFFMetadata().Keys {
		expected, ok := expectedKeyValues[key]
		if !ok {
			t.Errorf("unexpected key %v", key)
			continue
		}
		if val.Type == GTTagTypeDouble {
			if actual := val.AsDouble(); actual != expected.(float64) {
				t.Errorf("expected value %v for key %d but got %v", expected, key, actual)
			}

		}
		if val.KeyId != key {
			t.Errorf("expected key %d but got %d", key, val.KeyId)
		}
	}
}

func TestLasWithExtraBytes(t *testing.T) {
	f, err := os.Open("./testdata/extrabytes.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 1065
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		p, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		if p.PointDataRecordFormat != 3 {
			t.Errorf("expected point record format %d got %d", 3, p.PointDataRecordFormat)
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               4,
		SystemIdentifier:           "PDAL",
		GeneratingSoftware:         "PDAL 1.0.0.b1 (84d15e)",
		CreationDay:                53,
		CreationYear:               2015,
		HeaderSize:                 375,
		OffsetToPointData:          1389,
		NumberOfVLRs:               1,
		PointDataRecordFormat:      3,
		PointDataRecordLength:      61,
		LegacyNumberOfPointRecords: 1065,
		LegacyNumberOfPointByReturn: [5]uint32{
			925, 114, 21, 5, 0,
		},
		XScaleFactor:         0.01,
		YScaleFactor:         0.01,
		ZScaleFactor:         0.01,
		XOffset:              0,
		YOffset:              0,
		ZOffset:              0,
		MaxX:                 638982.55,
		MaxY:                 853535.43,
		MaxZ:                 586.38,
		MinX:                 635619.85,
		MinY:                 848899.7000000001,
		MinZ:                 406.59000000000003,
		NumberOfPointRecords: 1065,
		NumberOfPointsByReturn: [15]uint64{
			925, 114, 21, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
	if len(l.VLRs) != 1 {
		t.Errorf("expected 1 VLRs, got %d", len(l.VLRs))
	}
	v := l.VLRs[0]
	if v.RecordID != 4 {
		t.Errorf("expected VLR with record id %d, got %d", 4, v.RecordID)
	}
	if v.UserID != "LASF_Spec" {
		t.Errorf("expected VLR with user id %s, got %s", "LASF_Spec", v.UserID)
	}
	if v.RecordLengthAfterHeader != 960 {
		t.Errorf("expected VLR with length %d, got %d", 960, v.RecordLengthAfterHeader)
	}
	if len(v.Data) != 960 {
		t.Errorf("expected 960 bytes, got %v", len(v.Data))
	}
}

func TestLasWithLotsOfVLRs(t *testing.T) {
	f, err := os.Open("./testdata/lots_of_vlr.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 1
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		p, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		if p.PointDataRecordFormat != 1 {
			t.Errorf("expected point record format %d got %d", 3, p.PointDataRecordFormat)
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		ProjectIdGUID1:             8,
		ProjectIdGUID2:             30,
		ProjectIdGUID3:             2001,
		ProjectIdGUID4:             "ME_HD_1A",
		VersionMajor:               1,
		VersionMinor:               1,
		SystemIdentifier:           "",
		GeneratingSoftware:         "Merrick LiDAR Processing System",
		CreationDay:                1,
		CreationYear:               2002,
		HeaderSize:                 227,
		OffsetToPointData:          81891,
		NumberOfVLRs:               390,
		PointDataRecordFormat:      1,
		PointDataRecordLength:      28,
		LegacyNumberOfPointRecords: 1,
		LegacyNumberOfPointByReturn: [5]uint32{
			1, 0, 0, 0, 0,
		},
		XScaleFactor: 0.001,
		YScaleFactor: 0.001,
		ZScaleFactor: 0.001,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         715001.346,
		MinX:         715001.346,
		MaxY:         839349.171,
		MinY:         839349.171,
		MaxZ:         17.275000000000002,
		MinZ:         17.275000000000002,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
	if len(l.VLRs) != 390 {
		t.Errorf("expected 390 VLRs, got %d", len(l.VLRs))
	}
	v := l.VLRs[0]
	if v.RecordID != 101 {
		t.Errorf("expected VLR with record id %d, got %d", 101, v.RecordID)
	}
	if v.UserID != "Merrick" {
		t.Errorf("expected VLR with user id %s, got %s", "Merrick", v.UserID)
	}
	if v.RecordLengthAfterHeader != 342 {
		t.Errorf("expected VLR with length %d, got %d", 342, v.RecordLengthAfterHeader)
	}
	if len(v.Data) != 342 {
		t.Errorf("expected 342 bytes, got %v", len(v.Data))
	}
	v = l.VLRs[389]
	if v.RecordID != 34736 {
		t.Errorf("expected VLR with record id %d, got %d", 34736, v.RecordID)
	}
	if v.UserID != "LASF_Projection" {
		t.Errorf("expected VLR with user id %s, got %s", "LASF_Projection", v.UserID)
	}
	if v.RecordLengthAfterHeader != 40 {
		t.Errorf("expected VLR with length %d, got %d", 40, v.RecordLengthAfterHeader)
	}
	if len(v.Data) != 40 {
		t.Errorf("expected 40 bytes, got %v", len(v.Data))
	}
}

func TestLasSyntheticTest(t *testing.T) {
	f, err := os.Open("./testdata/synthetic_test.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedCount := 1
	if actual := l.NumberOfPoints(); actual != uint64(expectedCount) {
		t.Errorf("expected %d points, got %d", expectedCount, actual)
	}
	actualCount := 0
	for l.HasNext() {
		p, err := l.Next()
		if err != nil {
			if err != io.EOF {
				t.Errorf("unexpected error: %v", err)
			}
			break
		}
		if p.PointDataRecordFormat != 3 {
			t.Errorf("expected point record format %d got %d", 3, p.PointDataRecordFormat)
		}
		if actual := p.X; actual != 1.0 {
			t.Errorf("expected X: %f got %f", 1.0, actual)
		}
		if actual := p.Y; actual != 2.0 {
			t.Errorf("expected Y: %f got %f", 2.0, actual)
		}
		if actual := p.Z; actual != 3.0 {
			t.Errorf("expected Z: %f got %f", 3.0, actual)
		}
		if actual := len(p.ClassificationFlags()); actual != 1 {
			t.Errorf("expected 1 classification flags, got %d", actual)
		}
		if actual := p.ClassificationFlags()[0]; actual != ClassificationSynthetic {
			t.Errorf("expected ClassificationSynthetic, got %d", actual)
		}
		if actual := p.EdgeOfFlightLineFlag(); actual != EOFNormal {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", EOFNormal, actual)
		}
		if actual := p.ScanDirectionFlag(); actual != ScanDirectionNegative {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", ScanDirectionNegative, actual)
		}
		if actual := p.ScannerChannel(); actual != 0 {
			t.Errorf("expected EdgeOfFlightLineFlag %d, got %d", 0, actual)
		}
		if actual := p.ReturnNumber(); actual != 1 {
			t.Errorf("expected return number %d, got %d", 1, actual)
		}
		if actual := p.NumberOfReturns(); actual != 1 {
			t.Errorf("%d,expected number of returns %d, got %d", actualCount, 1, actual)
		}
		actualCount++
	}
	if actualCount != expectedCount {
		t.Errorf("expected to read exactly %d points, but read %d", expectedCount, actualCount)
	}

	expectedHeader := LasHeader{
		FileSignature:              "LASF",
		VersionMajor:               1,
		VersionMinor:               2,
		SystemIdentifier:           "PDAL",
		GeneratingSoftware:         "PDAL 2.0.0 (c85456)",
		CreationDay:                15,
		CreationYear:               2020,
		HeaderSize:                 227,
		OffsetToPointData:          227,
		NumberOfVLRs:               0,
		PointDataRecordFormat:      3,
		PointDataRecordLength:      34,
		LegacyNumberOfPointRecords: 1,
		LegacyNumberOfPointByReturn: [5]uint32{
			1, 0, 0, 0, 0,
		},
		XScaleFactor: 0.01,
		YScaleFactor: 0.01,
		ZScaleFactor: 0.01,
		XOffset:      0,
		YOffset:      0,
		ZOffset:      0,
		MaxX:         1,
		MinX:         1,
		MaxY:         2,
		MinY:         2,
		MaxZ:         3,
		MinZ:         3,
	}
	if expectedHeader != l.Header {
		t.Errorf("expected header \n%v, got \n%v", expectedHeader, l.Header)
	}
	if len(l.VLRs) != 0 {
		t.Errorf("expected 0 VLRs, got %d", len(l.VLRs))
	}
}

func TestLasEPSG4047(t *testing.T) {
	f, err := os.Open("./testdata/test_epsg_4047.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedEpsg := 4047
	expectedWkt := `GEOGCS["Unspecified datum based upon the GRS 1980 Authalic Sphere",DATUM["Not_specified_based_on_GRS_1980_Authalic_Sphere",SPHEROID["GRS 1980 Authalic Sphere",6371007,0,AUTHORITY["EPSG","7048"]],AUTHORITY["EPSG","6047"]],PRIMEM["Greenwich",0,AUTHORITY["EPSG","8901"]],UNIT["degree",0.0174532925199433,AUTHORITY["EPSG","9122"]],AUTHORITY["EPSG","4047"]]`
	if actual := l.GeoTIFFMetadata().Keys[2048].AsShort(); actual != uint16(expectedEpsg) {
		t.Errorf("expected EPSG %d in geotiff keys but got %d", expectedEpsg, actual)
	}
	if actual := l.WKT().CoordinateSystem; actual != expectedWkt {
		t.Errorf("expected WKT %s got %s", expectedWkt, actual)
	}
	if actual := l.CRS(); actual != expectedWkt {
		t.Errorf("expected CRS %s got %s", expectedWkt, actual)
	}
}

func TestLasUTM16(t *testing.T) {
	f, err := os.Open("./testdata/test_utm16.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedEpsg := 26916
	if actual := l.GeoTIFFMetadata().Keys[3072].AsShort(); actual != uint16(expectedEpsg) {
		t.Errorf("expected EPSG %d in geotiff keys but got %d", expectedEpsg, actual)
	}
	if actual := l.CRS(); actual != "EPSG:26916" {
		t.Errorf("expected CRS %s got %s", "EPSG:26916", actual)
	}
}

func TestLasUTM17(t *testing.T) {
	f, err := os.Open("./testdata/test_utm17.las")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := NewLas(f)
	if err != nil {
		t.Fatal(err)
	}
	expectedEpsg := 32617
	if actual := l.GeoTIFFMetadata().Keys[3072].AsShort(); actual != uint16(expectedEpsg) {
		t.Errorf("expected EPSG %d in geotiff keys but got %d", expectedEpsg, actual)
	}
	if actual := l.CRS(); actual != "EPSG:32617" {
		t.Errorf("expected CRS %s got %s", "EPSG:32617", actual)
	}
}
