package proj4_coordinate_converter

import (
	"bufio"
	"errors"
	"log"
	"math"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/tools"
	proj "github.com/xeonx/proj4"
)

const toRadians = math.Pi / 180
const toDeg = 180 / math.Pi

// proj4 wants coordinates in slices, to avoid always allocating one in the heap
// everytime is needed a buffer protected by a mutex is used
type floatBuffer struct {
	buf []float64
	sync.Mutex
}

type proj4CoordinateConverter struct {
	EpsgDatabase map[int]*epsgProjection
	// proj4 wants coordinates in slices, to avoid always allocating one
	// a set of buffers is used, with a synchronized index that stores the next available free buffer
	buffers   []*floatBuffer
	nextIndex int
	sync.Mutex
}

func NewProj4CoordinateConverter() converters.CoordinateConverter {
	exPath := tools.GetRootFolder()

	// Set path for retrieving projection assets data
	proj.SetFinder([]string{path.Join(exPath, "assets", "share")})

	// Initialization of EPSG Proj4 database
	file := path.Join(exPath, "assets", "epsg_projections.txt")

	return &proj4CoordinateConverter{
		EpsgDatabase: *loadEPSGProjectionDatabase(file),
		nextIndex:    0,
		buffers:      make([]*floatBuffer, 10*runtime.NumCPU()),
	}
}

func (cc *proj4CoordinateConverter) getFloatBuffer() *floatBuffer {
	cc.Lock()
	defer cc.Unlock()
	if cc.buffers[cc.nextIndex] == nil {
		cc.buffers[cc.nextIndex] = &floatBuffer{buf: make([]float64, 1)}
	}
	buf := cc.buffers[cc.nextIndex]
	cc.nextIndex += 1
	if cc.nextIndex >= len(cc.buffers) {
		cc.nextIndex = 0
	}
	return buf
}

func loadEPSGProjectionDatabase(databasePath string) *map[int]*epsgProjection {
	file := tools.OpenFileOrFail(databasePath)
	defer func() { _ = file.Close() }()

	var epsgDatabase = make(map[int]*epsgProjection)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		record := scanner.Text()
		code, projection := parseEPSGProjectionDatabaseRecord(record)
		epsgDatabase[code] = projection
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return &epsgDatabase
}

func parseEPSGProjectionDatabaseRecord(databaseRecord string) (int, *epsgProjection) {
	tokens := strings.Split(databaseRecord, "\t")
	code, err := strconv.Atoi(strings.Replace(tokens[0], "EPSG:", "", -1))
	if err != nil {
		log.Fatal("error while parsing the epsg projection file", err)
	}
	desc := tokens[1]
	proj4 := tokens[2]

	return code, &epsgProjection{
		EpsgCode:    code,
		Description: desc,
		Proj4:       proj4,
	}
}

// Converts the given coordinate from the given source Srid to the given target srid.
func (cc *proj4CoordinateConverter) ConvertCoordinateSrid(sourceSrid int, targetSrid int, coord geometry.Coordinate) (geometry.Coordinate, error) {
	if sourceSrid == targetSrid {
		return coord, nil
	}

	src, err := cc.initProjection(sourceSrid)
	if err != nil {
		return coord, err
	}

	dst, err := cc.initProjection(targetSrid)
	if err != nil {
		return coord, err
	}

	var converted, result = cc.executeConversion(&coord, src, dst)

	return *converted, result
}

// Converts the generic bounding box bounds values from the given input srid to a EPSG:4326 srid (in radians)
// and returns a float64 array containing xMin, yMin, xMax, yMax, zMin, zMax. Z values are left unchanged
func (cc *proj4CoordinateConverter) Convert2DBoundingboxToWGS84Region(bbox *geometry.BoundingBox, srid int) (*geometry.BoundingBox, error) {
	z := float64(0)
	projLowCorn := geometry.Coordinate{
		X: bbox.Xmin,
		Y: bbox.Ymin,
		Z: z,
	}
	projUppCorn := geometry.Coordinate{
		X: bbox.Xmax,
		Y: bbox.Ymax,
		Z: z,
	}
	w84lc, err := cc.ConvertCoordinateSrid(srid, 4326, projLowCorn)
	if err != nil {
		return nil, nil
	}
	w84uc, err := cc.ConvertCoordinateSrid(srid, 4326, projUppCorn)
	if err != nil {
		return nil, nil
	}

	return geometry.NewBoundingBox(w84lc.X*toRadians, w84lc.Y*toRadians, w84uc.X*toRadians, w84uc.Y*toRadians, bbox.Zmin, bbox.Zmax), nil
}

// Converts the input coordinate from the given srid to EPSG:4326 srid
func (cc *proj4CoordinateConverter) ConvertToWGS84Cartesian(coord geometry.Coordinate, sourceSrid int) (geometry.Coordinate, error) {
	if sourceSrid == 4978 {
		return coord, nil
	}

	res, err := cc.ConvertCoordinateSrid(sourceSrid, 4326, coord)
	if err != nil {
		return coord, err
	}
	res2, err := cc.ConvertCoordinateSrid(4329, 4978, res)
	return res2, err
}

// Releases all projection objects from memory
func (cc *proj4CoordinateConverter) Cleanup() {
	for _, val := range cc.EpsgDatabase {
		if val.Projection != nil {
			val.Projection.Close()
		}
	}
}

func (cc *proj4CoordinateConverter) executeConversion(coord *geometry.Coordinate, sourceProj *proj.Proj, destinationProj *proj.Proj) (*geometry.Coordinate, error) {
	xBuf := cc.getFloatBuffer()
	yBuf := cc.getFloatBuffer()
	xBuf.Lock()
	yBuf.Lock()
	defer xBuf.Unlock()
	defer yBuf.Unlock()

	x := xBuf.buf
	y := yBuf.buf
	var z []float64 = nil
	x[0] = getCoordinateInRadiansFromSridFormat(coord.X, sourceProj)
	y[0] = getCoordinateInRadiansFromSridFormat(coord.Y, sourceProj)

	if !math.IsNaN(coord.Z) {
		zBuf := cc.getFloatBuffer()
		zBuf.Lock()
		defer zBuf.Unlock()
		z = zBuf.buf
		z[0] = coord.Z
	}

	var err = proj.TransformRaw(sourceProj, destinationProj, x, y, z)

	var converted = geometry.Coordinate{
		X: getCoordinateFromRadiansToSridFormat(x[0], destinationProj),
		Y: getCoordinateFromRadiansToSridFormat(y[0], destinationProj),
		Z: extractZPointerIfPresent(z),
	}

	return &converted, err
}

// Returns the input coordinate expressed in the given srid converting it into radians if necessary
func getCoordinateInRadiansFromSridFormat(coord float64, srid *proj.Proj) float64 {
	var radians = coord

	if srid.IsLatLong() {
		radians = coord * toRadians
	}

	return radians
}

func extractZPointerIfPresent(zContainer []float64) float64 {
	if zContainer != nil {
		return zContainer[0]
	}

	return math.NaN()
}

// Returns the input coordinate expressed in the given srid converting it into radians if necessary
func getCoordinateFromRadiansToSridFormat(coord float64, srid *proj.Proj) float64 {
	var angle = coord

	if srid.IsLatLong() {
		angle = coord * toDeg
	}

	return angle
}

// Returns the projection corresponding to the given EPSG code, storing it in the relevant EpsgDatabase entry for caching
func (cc *proj4CoordinateConverter) initProjection(code int) (*proj.Proj, error) {
	val, ok := cc.EpsgDatabase[code]
	if !ok {
		return &proj.Proj{}, errors.New("epsg code not found")
	} else if val.Projection == nil {
		projection, err := proj.InitPlus(val.Proj4)
		if err != nil {
			return &proj.Proj{}, errors.New("unable to init projection")
		}
		val.Projection = projection
	}
	return val.Projection, nil
}
