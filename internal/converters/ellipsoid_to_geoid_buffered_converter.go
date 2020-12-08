package converters

import (
	"math"
	"sync"
)

// Represent minimal data necessary to provide an efficient, cache-based solution for the massive geodetic to ellipsoidic height conversion
type EllipsoidToGeoidBufferedConverter struct {
	SourceSrid         int
	CellSize           float64
	GeoidHeightMap     sync.Map
	ElevationConverter EllipsoidToGeoidZConverter
}

// Inits a new instance of EllipsoidToGeoidBufferedConverter for the given Srid and with given caching cell size. To all points with X,Y coordinates
// falling inside a square cell with side equal to cellSize will be applied the same, eventually cached, elevation transformation.
// Choosing a small value for cell size improves the accuracy but increases computation times. Cell sizes approximately
// equivalent to 1m are acceptable approximations
func NewElevationConverterBuffer(srid int, cellSize float64, elevationConverter EllipsoidToGeoidZConverter) *EllipsoidToGeoidBufferedConverter {
	return &EllipsoidToGeoidBufferedConverter{
		SourceSrid:         srid,
		CellSize:           cellSize,
		ElevationConverter: elevationConverter,
	}
}

func (elevationConverterBuffer *EllipsoidToGeoidBufferedConverter) GetConvertedElevation(lon, lat, inputElevation float64) (float64, error) {
	x := elevationConverterBuffer.getCellIndex(lon)
	y := elevationConverterBuffer.getCellIndex(lat)

	yMap, yMapPresent := elevationConverterBuffer.GeoidHeightMap.Load(x)

	if !yMapPresent {
		var temp sync.Map
		elevationConverterBuffer.GeoidHeightMap.Store(x, &temp)
		yMap = &temp
	}

	yVal, yValPresent := yMap.(*sync.Map).Load(y)
	if yValPresent {
		// return cached result
		return yVal.(float64) + inputElevation, nil
	}

	// else compute offset and store in cache
	off, err := computeAndStoreInMap(x, y, elevationConverterBuffer, yMap.(*sync.Map))

	if err != nil {
		return 0, err
	}

	// return offset + input elevation
	return off + inputElevation, nil
}

func (elevationConverterBuffer *EllipsoidToGeoidBufferedConverter) getCellIndex(dimensionValue float64) int {
	return int(math.Floor(dimensionValue / elevationConverterBuffer.CellSize))
}

func (elevationConverterBuffer *EllipsoidToGeoidBufferedConverter) getCellCenter(x, y int) (float64, float64) {
	return float64(x)*elevationConverterBuffer.CellSize + elevationConverterBuffer.CellSize/2, float64(y)*elevationConverterBuffer.CellSize + elevationConverterBuffer.CellSize/2
}

func computeAndStoreInMap(x, y int, elevationConverterBuffer *EllipsoidToGeoidBufferedConverter, yMap *sync.Map) (float64, error) {
	off, err := getCellEllipsoidToGeoidOffset(x, y, elevationConverterBuffer)

	if err != nil {
		return 0, err
	}

	yMap.Store(y, off)
	return off, nil
}

func getCellEllipsoidToGeoidOffset(x, y int, elevationConverterBuffer *EllipsoidToGeoidBufferedConverter) (float64, error) {
	cX, cY := elevationConverterBuffer.getCellCenter(x, y)

	return (elevationConverterBuffer.ElevationConverter).GetEllipsoidToGeoidZOffset(cX, cY, elevationConverterBuffer.SourceSrid)

}
