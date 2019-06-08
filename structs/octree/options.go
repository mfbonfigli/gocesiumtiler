package octree

type TilerOptions struct {
	Input                  string
	Output                 string
	Srid                   int
	ZOffset                float64
	MaxNumPointsPerNode    int32
	EnableGeoidZCorrection bool
	FolderProcessing       bool
	Recursive              bool
	SubfolderPrefix        string
	Silent                 bool
	Strategy               LoaderStrategy
}
