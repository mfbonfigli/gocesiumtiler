package converters

import (
	"math"
	"sync"
)

// Represent minimal data necessary to provide an efficient solution for the massive geodetic to ellipsoidic height conversion
type ElevationFixer struct {
	SourceSrid     int
	CellSize       float64
	geoidHeightMap sync.Map
}

// Inits a new instance of ElevationFixer for the given Srid and with given caching cell size. To all points with X,Y coordinates
// falling inside a square cell with side equal to cellSize will be applied the same, eventually cached, elevation transformation.
// Choosing a small value for cell size improves the accuracy but increases computation times. Cell sizes approximately
// equivalent to 1m are acceptable approximations
func NewElevationFixer(srid int, cellSize float64) *ElevationFixer {
	return &ElevationFixer{
		SourceSrid: srid,
		CellSize:   cellSize,
	}
}

func (elevFix *ElevationFixer) getCellIndex(dimensionValue float64) int {
	return int(math.Floor(dimensionValue / elevFix.CellSize))
}

func (elevFix *ElevationFixer) getCellCenter(x, y int) (float64, float64) {
	return float64(x)*elevFix.CellSize + elevFix.CellSize/2, float64(y)*elevFix.CellSize + elevFix.CellSize/2
}

func (elevFix *ElevationFixer) GetCorrectedElevation(lon, lat, originalElevation float64) (float64, error) {
	x := elevFix.getCellIndex(lon)
	y := elevFix.getCellIndex(lat)
	valX, okX := elevFix.geoidHeightMap.Load(x)
	if okX {
		valY, okY := valX.(*sync.Map).Load(y)
		if okY {
			return valY.(float64), nil
		} else {
			cX, cY := elevFix.getCellCenter(x, y)
			off, err := GetEllipsoidToGeoidOffset(cX, cY, elevFix.SourceSrid)
			if err != nil {
				return 0, err
			}
			valX.(*sync.Map).Store(y, off)
			return off + originalElevation, nil
		}
	} else {
		var mappa sync.Map
		elevFix.geoidHeightMap.Store(x, &mappa)
		cX, cY := elevFix.getCellCenter(x, y)
		off, err := GetEllipsoidToGeoidOffset(cX, cY, elevFix.SourceSrid)
		if err != nil {
			return 0, err
		}
		mappa.Store(y, off)
		return off + originalElevation, nil
	}
}
