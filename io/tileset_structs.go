package io

type Asset struct {
	Version string `json:"version"`
}

type Content struct {
	Url string `json:"uri"`
}

type BoundingVolume struct {
	Region []float64 `json:"region"`
}

type Child struct {
	Content        Content        `json:"content"`
	BoundingVolume BoundingVolume `json:"boundingVolume"`
	GeometricError float64        `json:"geometricError"`
	Refine         string         `json:"refine"`
}

type Root struct {
	Children       []Child        `json:"children"`
	Content        Content        `json:"content"`
	BoundingVolume BoundingVolume `json:"boundingVolume"`
	GeometricError float64        `json:"geometricError"`
	Refine         string         `json:"refine"`
}

type Tileset struct {
	Asset          Asset   `json:"asset"`
	GeometricError float64 `json:"geometricError"`
	Root           Root    `json:"root"`
}
