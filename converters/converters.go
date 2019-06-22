package converters

import (
	"errors"
	"github.com/xeonx/proj4"
	"go_cesium_tiler/structs"
	"go_cesium_tiler/structs/octree"
	"math"
)

const toRadians = math.Pi / 180
const toDeg = 180 / math.Pi

// Converts the given coordinate from the given source Srid to the given target srid.
func Convert(sourceSrid int, targetSrid int, coord structs.Coordinate) (structs.Coordinate, error) {
	if sourceSrid == targetSrid {
		return coord, nil
	}

	src, err := initProjection(sourceSrid)
	if err != nil {
		return coord, err
	}

	dst, err := initProjection(targetSrid)
	if err != nil {
		return coord, err
	}

	var x, y, z []float64
	x = []float64{*coord.X}
	y = []float64{*coord.Y}
	if src.IsLatLong() {
		xv := *coord.X * toRadians
		x = []float64{xv}
		yv := *coord.Y * toRadians
		y = []float64{yv}
	}
	if coord.Z != nil {
		z = []float64{*coord.Z}
	}
	err = proj.TransformRaw(src, dst, x, y, z)

	xConv := &x[0]
	if dst.IsLatLong() {
		xc := *xConv * toDeg
		xConv = &xc
	}

	yConv := &y[0]
	if dst.IsLatLong() {
		yc := *yConv * toDeg
		yConv = &yc
	}

	var zConv *float64
	if z != nil {
		zConv = &z[0]
	}

	return structs.Coordinate{
		X: xConv,
		Y: yConv,
		Z: zConv,
	}, err
}

// Converts the input coordinate from the given srid to EPSG:4326 srid
func ConvertToWGS84Cartesian(coord structs.Coordinate, sourceSrid int) (structs.Coordinate, error) {
	res, err := Convert(sourceSrid, 4326, coord)
	if err != nil {
		return coord, err
	}
	res2, err := Convert(4329, 4978, res)
	return res2, err
}

// Converts the generic bounding box bounds values from the given input srid to a EPSG:4326 srid (in radians)
// and returns a float64 array containing xMin, yMin, xMax, yMax, zMin, zMax. Z values are left unchanged
func Convert2DBoundingboxToWGS84Region(bbox *octree.BoundingBox, srid int) ([]float64, error) {
	z := float64(0)
	projLowCorn := structs.Coordinate{
		X: &bbox.Xmin,
		Y: &bbox.Ymin,
		Z: &z,
	}
	projUppCorn := structs.Coordinate{
		X: &bbox.Xmax,
		Y: &bbox.Ymax,
		Z: &z,
	}
	w84lc, err := Convert(srid, 4326, projLowCorn)
	if err != nil {
		return nil, nil
	}
	w84uc, err := Convert(srid, 4326, projUppCorn)
	if err != nil {
		return nil, nil
	}

	return []float64{*w84lc.X * toRadians, *w84lc.Y * toRadians, *w84uc.X * toRadians, *w84uc.Y * toRadians, bbox.Zmin, bbox.Zmax}, nil
}

// Returns the distance in meters between the geoid and the ellipsoid height at the given latitude and longitude
func GetEllipsoidToGeoidOffset(lat, lon float64, sourceSrid int) (float64, error) {
	coordEPSG4326, err := Convert(sourceSrid, 4326, structs.Coordinate{X: &lat, Y: &lon, Z: nil})
	if err != nil {
		return 0, err
	}
	off := GH.HeightOffset(*coordEPSG4326.X, *coordEPSG4326.Y, 0)
	return off, err
}

// Returns the projection corresponding to the given EPSG code, storing it in the relevant EpsgDatabase entry for caching
func initProjection(code int) (*proj.Proj, error) {
	val, ok := EpsgDatabase[code]
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

func DeallocateProjection(code int) {
	val, ok := EpsgDatabase[code]
	if ok {
		val.Projection.Close()
		EpsgDatabase[code].Projection = nil;
	}
}

func DeallocateProjections() {
	for _, val := range EpsgDatabase {
		if val.Projection != nil {
			val.Projection.Close()
		}
	}
}
