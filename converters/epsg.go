package converters

import proj "github.com/xeonx/proj4"

// Represents a EPSG reference system and stores the relevant projection object for caching reasons
type EpsgProjection struct {
	EpsgCode    int
	Description string
	Proj4       string
	Projection  *proj.Proj
}
