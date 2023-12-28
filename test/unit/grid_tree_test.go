package unit

import (
	"testing"

	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree/grid_tree"
)

type mockLasReader struct {
	ptProv func(int) (x, y, z float64, r, g, b, in, cls uint8)
	nPts   int
	srid   int
}

func (m *mockLasReader) NumberOfPoints() int {
	return m.nPts
}

func (m *mockLasReader) GetPointAt(i int) (x, y, z float64, r, g, b, in, cls uint8) {
	return m.ptProv(i)
}

func (m *mockLasReader) GetSrid() int {
	return m.srid
}

type mockCoordinateConverter struct{}

func (m *mockCoordinateConverter) ConvertCoordinateSrid(sourceSrid int, targetSrid int, coord geometry.Coordinate) (geometry.Coordinate, error) {
	return coord, nil
}

func (m *mockCoordinateConverter) Convert2DBoundingboxToWGS84Region(bbox *geometry.BoundingBox, srid int, offX, offY, offZ float64) (*geometry.BoundingBox, error) {
	return bbox, nil
}

func (m *mockCoordinateConverter) ConvertToWGS84Cartesian(coord geometry.Coordinate, sourceSrid int) (geometry.Coordinate, error) {
	return coord, nil
}

func (m *mockCoordinateConverter) Cleanup() {}

func TestTreeAddPointSuccess(t *testing.T) {
	tree := grid_tree.NewGridTree(
		&mockCoordinateConverter{},
		&mockElevationCorrector{},
		5.0,
		0.1,
	)

	x := 14.0
	y := 41.0
	z := 1.2
	r := uint8(4)
	g := uint8(5)
	b := uint8(6)
	i := uint8(7)
	c := uint8(8)

	coord := &geometry.Coordinate{
		X: x,
		Y: y,
		Z: z,
	}

	tree.AddPoint(coord, r, g, b, i, c, 4326)

	point, hasMore := tree.(*grid_tree.GridTree).Loader.GetNext()

	if hasMore == true {
		t.Errorf("Only one point loaded, GetNext should return false")
	}

	if point.X != float32(x) || point.Y != float32(y) || point.Z != float32(2.4) ||
		point.R != r || point.G != g || point.B != b ||
		point.Intensity != i || point.Classification != c {
		t.Errorf("Wrong point data found")
	}
}

func TestTreeBuildSuccess(t *testing.T) {
	tree := grid_tree.NewGridTree(
		&mockCoordinateConverter{},
		&mockElevationCorrector{},
		5.0,
		0.1,
	)

	x := 14.0
	y := 41.0
	z := 3.0
	r := uint8(4)
	g := uint8(5)
	b := uint8(6)
	i := uint8(7)
	c := uint8(8)

	mockReader := &mockLasReader{
		ptProv: func(_ int) (float64, float64, float64, uint8, uint8, uint8, uint8, uint8) {
			return x, y, z, r, g, b, i, c
		},
		nPts: 1,
		srid: 4326,
	}

	//tree.AddPoint(coord, r, g, b, i, c, 4326)

	err := tree.Build(mockReader)

	if err != nil {
		t.Errorf("Unexpected error occurred while building the tree: %s", err)
	}

	if !tree.IsBuilt() {
		t.Errorf("Tree signals that it is not build but should have been")
	}

	if len(tree.GetRootNode().GetPoints()) != 1 {
		t.Errorf("Tree root node does not contain exactly one node but %d instead", len(tree.GetRootNode().GetPoints()))
	}
}

func TestGetRootNode(t *testing.T) {
	tree := grid_tree.NewGridTree(
		&mockCoordinateConverter{},
		&mockElevationCorrector{},
		5.0,
		0.1,
	)

	x := 14.0
	y := 41.0
	z := 3.0
	r := uint8(4)
	g := uint8(5)
	b := uint8(6)
	i := uint8(7)
	c := uint8(8)

	//tree.AddPoint(coord, r, g, b, i, c, 4326)

	mockReader := &mockLasReader{
		ptProv: func(_ int) (float64, float64, float64, uint8, uint8, uint8, uint8, uint8) {
			return x, y, z, r, g, b, i, c
		},
		nPts: 1,
		srid: 4326,
	}

	err := tree.Build(mockReader)

	if err != nil {
		t.Errorf("Unexpected error occurred while building the tree: %s", err)
	}

	node := tree.GetRootNode()

	if node == nil {
		t.Errorf("Nil root node returned")
	}

	if len(node.GetPoints()) != 1 {
		t.Errorf("Root Node has wrong number of points")
	}
}

// TODO add test to evaluate safety against race conditions while adding points,
//  especially check against gridCell being correctly write locked when points slice is edited
