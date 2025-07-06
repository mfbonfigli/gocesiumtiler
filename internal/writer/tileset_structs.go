package writer

import "github.com/mfbonfigli/gocesiumtiler/v2/version"

type Asset struct {
	Version version.TilesetVersion `json:"version"`
}

type Content struct {
	Url string `json:"uri"`
}

type BoundingVolume struct {
	Box [12]float64 `json:"box"`
}

type Child struct {
	Content        *Content       `json:"content,omitempty"`
	BoundingVolume BoundingVolume `json:"boundingVolume"`
	GeometricError float64        `json:"geometricError"`
	Refine         string         `json:"refine"`
	Children       []*Child       `json:"children,omitempty"`
}

type Root struct {
	Children       []*Child       `json:"children,omitempty"`
	Content        *Content       `json:"content,omitempty"`
	BoundingVolume BoundingVolume `json:"boundingVolume"`
	GeometricError float64        `json:"geometricError"`
	Refine         string         `json:"refine"`
	Transform      *[16]float64   `json:"transform,omitempty"`
}

type Tileset struct {
	Asset          Asset   `json:"asset"`
	GeometricError float64 `json:"geometricError"`
	Root           Root    `json:"root"`
}
