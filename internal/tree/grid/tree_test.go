package grid

import (
	"context"
	"math"
	"testing"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/las"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils/test"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/model"
)

func TestNewGridTree(t *testing.T) {
	tree := NewTree(
		WithGridSize(11.5),
		WithMaxDepth(12),
		WithMinPointsPerChildren(1),
		WithLoadWorkersNumber(2),
	)
	if tree.RootNode() != tree {
		t.Errorf("the root node of a tree is the tree itself but it was not")
	}
	if tree.IsRoot() != true {
		t.Errorf("the tree object should be a root node")
	}
	if tree.config.maxDepth != 12 {
		t.Errorf("expected maxDepth %d but got %d", 12, tree.config.maxDepth)
	}
	if tree.depth != 0 {
		t.Errorf("expected depth %d but got %d", 0, tree.depth)
	}
	if tree.gridSize != 11.5 {
		t.Errorf("expected gridSize %f but got %f", 11.5, tree.gridSize)
	}
	if tree.config.loadWorkersNumber != 2 {
		t.Errorf("expected loadWorkersNumber %d but got %d", 2, tree.config.loadWorkersNumber)
	}
}

func TestNewGridTreeDefaults(t *testing.T) {
	tree := NewTree()
	if tree.RootNode() != tree {
		t.Errorf("the root node of a tree is the tree itself but it was not")
	}
	if tree.IsRoot() != true {
		t.Errorf("the tree object should be a root node")
	}
	if tree.config.maxDepth != 10 {
		t.Errorf("expected maxDepth %d but got %d", 10, tree.config.maxDepth)
	}
	if tree.depth != 0 {
		t.Errorf("expected depth %d but got %d", 0, tree.depth)
	}
	if tree.gridSize != 1.0 {
		t.Errorf("expected gridSize %f but got %f", 1.0, tree.gridSize)
	}
	if tree.config.loadWorkersNumber != 1 {
		t.Errorf("expected loadWorkersNumber %d but got %d", 1, tree.config.loadWorkersNumber)
	}
	if tree.config.minPointsPerChildren != 2000 {
		t.Errorf("expected minPointsPerChildren %d but got %d", 2000, tree.config.minPointsPerChildren)
	}
}

func TestGridTreeLoad(t *testing.T) {
	// the grid size is kept big intentionally so that we have at most 1 point per octant during the tests
	tree := NewTree(WithGridSize(1000000), WithMaxDepth(3), WithMinPointsPerChildren(1), WithLoadWorkersNumber(1))
	reader := &las.MockLasReader{
		CRS: "EPSG:32633",
		Pts: []geom.Point64{
			{Vector: model.Vector{X: 432488.4714159001, Y: 4.705678720925195e+06, Z: 2.550538045727727}, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432466.58372129063, Y: 4.705686414284739e+06, Z: 4.457767479175496}, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432469.98167019564, Y: 4.705681849831438e+06, Z: 4.655027394763156}, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432456.1696575739, Y: 4.705683863015449e+06, Z: 1.7922372985151949}, R: 107, G: 114, B: 156, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432465.34641605255, Y: 4.705682466042181e+06, Z: 1.9543476118949155}, R: 165, G: 176, B: 206, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432471.7795208326, Y: 4.705664509499522e+06, Z: 1.9162578617315837}, R: 90, G: 136, B: 213, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432449.48655283387, Y: 4.705672817604579e+06, Z: 2.7424278273838762}, R: 51, G: 66, B: 87, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432457.86106384156, Y: 4.705682649635591e+06, Z: 2.081788046497537}, R: 80, G: 97, B: 123, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432466.3000286405, Y: 4.705680013894157e+06, Z: 1.9357872046388636}, R: 160, G: 169, B: 202, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432455.7336230836, Y: 4.705657784890012e+06, Z: 6.120847715554042}, R: 73, G: 78, B: 110, Intensity: 7, Classification: 3},
		},
	}

	expectedAbsolute := []geom.Point64{
		{Vector: model.Vector{X: 0, Y: 0, Z: 0}, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: -3.6806421, Y: -22.91469, Z: 1.9071872}, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 0.20831008, Y: -18.757929, Z: 2.104462}, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 0.6830035, Y: -32.712612, Z: -0.7583845}, R: 107, G: 114, B: 156, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 0.42609432, Y: -23.430492, Z: -0.5962334}, R: 165, G: 176, B: 206, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 16.958508, Y: -13.9041, Z: -0.6343179}, R: 90, G: 136, B: 213, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 12.744787, Y: -37.327072, Z: 0.19176799}, R: 51, G: 66, B: 87, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 1.5766444, Y: -30.83178, Z: -0.4688246}, R: 80, G: 97, B: 123, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 2.6704051, Y: -22.055632, Z: -0.6147895}, R: 160, G: 169, B: 202, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 26.432066, Y: -28.503836, Z: 3.5701911}, R: 73, G: 78, B: 110, Intensity: 7, Classification: 3},
	}

	expected := make([]model.Point, len(expectedAbsolute))
	for i, e := range expectedAbsolute {
		expected[i] = model.Point{
			X:              float32(e.X),
			Y:              float32(e.Y),
			Z:              float32(e.Z),
			R:              e.R,
			G:              e.G,
			B:              e.B,
			Intensity:      e.Intensity,
			Classification: e.Classification,
		}
	}
	conv := test.GetTestCoordinateConverterFactory()
	tree.Load(reader, conv, nil, context.TODO())
	// verify the points are stored
	cur := tree.pts
	for i := 0; i < len(reader.Pts); i++ {
		if cur.Pt != expected[i] {
			t.Errorf("expected pt %v got %v", expected[i], cur.Pt)
		}
		cur = cur.Next
	}

	// build
	err := tree.Build()
	if err != nil {
		t.Fatalf("unexpected error during tree build: %v", err)
	}
	root := tree.RootNode()
	if actual := root.TotalNumberOfPoints(); actual != 10 {
		t.Errorf("expected 10 points, got %d", actual)
	}
}

func TestGridTreeGeometricError(t *testing.T) {
	tree := NewTree(WithGridSize(1))
	expected := math.Sqrt(3) * 1.7
	if actual := tree.GeometricError(); actual != expected {
		t.Errorf("expected error %v, got %v", expected, actual)
	}
}

func TestGridTreeBuild(t *testing.T) {
	// the grid size is kept big intentionally so that we have at most 1 point per octant during the tests
	tree := NewTree(WithGridSize(1000000), WithMaxDepth(3), WithMinPointsPerChildren(1))
	reader := &las.MockLasReader{
		CRS: "EPSG:4978",
		Pts: []geom.Point64{
			{Vector: model.Vector{X: 0, Y: 0, Z: 0}, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: -1, Y: -1, Z: -1}, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 1, Y: 1, Z: 1}, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: -1, Y: -1, Z: 1}, R: 107, G: 114, B: 156, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 1, Y: 1, Z: -1}, R: 165, G: 176, B: 206, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: -1, Y: 1, Z: 1}, R: 90, G: 136, B: 213, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: -1, Y: 1, Z: -1}, R: 51, G: 66, B: 87, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 1, Y: -1, Z: 1}, R: 80, G: 97, B: 123, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 1, Y: -1, Z: -1}, R: 160, G: 169, B: 202, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 0.5, Y: 0.5, Z: 0.5}, R: 73, G: 78, B: 110, Intensity: 7, Classification: 3},
		},
	}

	expectedAbsolute := []geom.Point64{
		{Vector: model.Vector{X: 0, Y: 0, Z: 0}, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3}, // baseline
		{Vector: model.Vector{X: -1, Y: -1, Z: -1}, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 1, Y: 1, Z: 1}, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: -1, Y: -1, Z: 1}, R: 107, G: 114, B: 156, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 1, Y: 1, Z: -1}, R: 165, G: 176, B: 206, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: -1, Y: 1, Z: 1}, R: 90, G: 136, B: 213, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: -1, Y: 1, Z: -1}, R: 51, G: 66, B: 87, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 1, Y: -1, Z: 1}, R: 80, G: 97, B: 123, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 1, Y: -1, Z: -1}, R: 160, G: 169, B: 202, Intensity: 7, Classification: 3},
		{Vector: model.Vector{X: 0.5, Y: 0.5, Z: 0.5}, R: 73, G: 78, B: 110, Intensity: 7, Classification: 3},
	}
	expected := make([]model.Point, len(expectedAbsolute))
	for i, e := range expectedAbsolute {
		expected[i] = model.Point{
			X:              float32(e.X),
			Y:              float32(e.Y),
			Z:              float32(e.Z),
			R:              e.R,
			G:              e.G,
			B:              e.B,
			Intensity:      e.Intensity,
			Classification: e.Classification,
		}
	}
	conv := test.GetTestCoordinateConverterFactory()

	tree.Load(reader, conv, nil, context.TODO())
	// verify the points are stored
	cur := tree.pts
	for i := 0; i < len(reader.Pts); i++ {
		if cur.Pt != expected[i] {
			t.Errorf("expected pt %v got %v", expected[i], cur.Pt)
		}
		cur = cur.Next
	}

	// build
	err := tree.Build()
	if err != nil {
		t.Fatalf("unexpected error during tree build: %v", err)
	}

	root := tree.RootNode()
	if actual := root.NumberOfPoints(); actual != 1 {
		t.Errorf("expected 1 points, got %d", actual)
	}
	if actual := root.TotalNumberOfPoints(); actual != 10 {
		t.Errorf("expected 10 points, got %d", actual)
	}
	pt, err := root.Points().Next()
	if err != nil {
		t.Fatalf("unexpected error during point retrieval: %v", err)
	}
	if pt != expected[0] {
		t.Errorf("unexpected point returned, expected %v, got %v", expected[0], pt)
	}

	childExpectedMap := []model.Point{
		expected[1],
		expected[8],
		expected[6],
		expected[4],
		expected[3],
		expected[7],
		expected[5],
		expected[9],
	}
	for i := range uint8(8) {
		c := root.ChildrenAt(i)
		if c == nil {
			t.Logf("Child %d is nil", i)
			continue
		}
		if actual := c.NumberOfPoints(); actual != 1 {
			t.Errorf("expected 1 points, got %d", actual)
		}
		if i != 7 {
			if actual := c.TotalNumberOfPoints(); actual != 1 {
				t.Errorf("expected 1 point, got %d", actual)
			}
			if actual := c.IsLeaf(); actual != true {
				t.Errorf("expected point to be leaf, got %v", actual)
			}

		} else {
			if actual := c.TotalNumberOfPoints(); actual != 2 {
				t.Errorf("expected 2 points, got %d", actual)
			}
			if actual := c.IsLeaf(); actual != false {
				t.Errorf("expected point to NOT be leaf, got %v", actual)
			}
		}
		if actual := c.IsRoot(); actual != false {
			t.Errorf("expected point to NOT be root, got %v", actual)
		}
		pt, err := c.Points().Next()
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if pt != childExpectedMap[i] {
			t.Errorf("unexpected point returned for children %d, expected %v, got %v", i, childExpectedMap[i], pt)
		}
		if i == 7 {
			for i := 0; i < 8; i++ {
				if i == 7 {
					if n := c.ChildrenAt(uint8(i)).NumberOfPoints(); n != 1 {
						t.Errorf("expected 1 point but got %d", n)
					}
					pt, err := c.ChildrenAt(uint8(i)).Points().Next()
					if err != nil {
						t.Fatalf("unexpected error %v", err)
					}
					if pt != expected[2] {
						t.Errorf("unexpected point returned for children %d, expected %v, got %v", i, expected[2], pt)
					}
				} else {
					if c.ChildrenAt(uint8(i)) != nil {
						t.Errorf("expected no child, got one: %v", c.ChildrenAt(uint8(i)))
					}
				}
			}
		} else {
			for i := range 8 {
				if val := c.ChildrenAt(uint8(i)); val != nil {
					t.Errorf("expected no child, got one: %v", val)
				}
			}
		}
	}
	if !reader.CloseCalled {
		t.Errorf("expected reader to be closed but was not")
	}
}

func TestGetBoundingBoxRegion(t *testing.T) {
	// the grid size is kept big intentionally so that we have at most 1 point per octant during the tests
	tree := NewTree(WithGridSize(1000000), WithMaxDepth(3), WithMinPointsPerChildren(1))
	reader := &las.MockLasReader{
		CRS: "EPSG:32633",
		Pts: []geom.Point64{
			{Vector: model.Vector{X: 432488.4714159001, Y: 4.705678720925195e+06, Z: 2.550538045727727}, R: 160, G: 166, B: 203, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432466.58372129063, Y: 4.705686414284739e+06, Z: 4.457767479175496}, R: 186, G: 200, B: 237, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432469.98167019564, Y: 4.705681849831438e+06, Z: 4.655027394763156}, R: 156, G: 167, B: 204, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432456.1696575739, Y: 4.705683863015449e+06, Z: 1.7922372985151949}, R: 107, G: 114, B: 156, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432465.34641605255, Y: 4.705682466042181e+06, Z: 1.9543476118949155}, R: 165, G: 176, B: 206, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432471.7795208326, Y: 4.705664509499522e+06, Z: 1.9162578617315837}, R: 90, G: 136, B: 213, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432449.48655283387, Y: 4.705672817604579e+06, Z: 2.7424278273838762}, R: 51, G: 66, B: 87, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432457.86106384156, Y: 4.705682649635591e+06, Z: 2.081788046497537}, R: 80, G: 97, B: 123, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432466.3000286405, Y: 4.705680013894157e+06, Z: 1.9357872046388636}, R: 160, G: 169, B: 202, Intensity: 7, Classification: 3},
			{Vector: model.Vector{X: 432455.7336230836, Y: 4.705657784890012e+06, Z: 6.120847715554042}, R: 73, G: 78, B: 110, Intensity: 7, Classification: 3},
		},
	}

	conv := test.GetTestCoordinateConverterFactory()
	var err error
	if err != nil {
		t.Errorf("unexpected error while initializing the converter: %v", err)
	}
	err = tree.Load(reader, conv, nil, context.TODO())
	if err != nil {
		t.Fatalf("unexpected error during tree load: %v", err)
	}
	err = tree.Build()
	if err != nil {
		t.Fatalf("unexpected error during tree build: %v", err)
	}
	bbox := tree.BoundingBox()
	if err != nil {
		t.Fatalf("unexpected error during tree build: %v", err)
	}
	expected := geom.BoundingBox{
		Xmin: -3.680642,
		Xmax: 26.432066,
		Ymin: -37.327072,
		Ymax: 0.000000,
		Zmin: -0.758385,
		Zmax: 3.570191,
	}
	if diff, err := utils.CompareWithTolerance(bbox.Xmin, expected.Xmin, 1e-6); err != nil {
		t.Errorf("Xmin diff above threshold: %f, expected %f", diff, bbox.Xmin)
	}
	if diff, err := utils.CompareWithTolerance(bbox.Xmax, expected.Xmax, 1e-6); err != nil {
		t.Errorf("Xmax diff above threshold: %f, expected %f", diff, bbox.Xmax)
	}
	if diff, err := utils.CompareWithTolerance(bbox.Ymin, expected.Ymin, 1e-6); err != nil {
		t.Errorf("Ymin diff above threshold: %f, expected %f", diff, bbox.Ymin)
	}
	if diff, err := utils.CompareWithTolerance(bbox.Ymax, expected.Ymax, 1e-6); err != nil {
		t.Errorf("Ymax diff above threshold: %f, expected %f", diff, bbox.Ymax)
	}
	if diff, err := utils.CompareWithTolerance(bbox.Zmin, expected.Zmin, 1e-6); err != nil {
		t.Errorf("Zmin diff above threshold: %f, expected %f", diff, bbox.Zmin)
	}
	if diff, err := utils.CompareWithTolerance(bbox.Zmax, expected.Zmax, 1e-6); err != nil {
		t.Errorf("Zmax diff above threshold: %f, expected %f", diff, bbox.Zmax)
	}
}

func TestGridTreeBuildWithMaxPoints(t *testing.T) {
	// the grid size is kept big intentionally so that we have at most 1 point per octant during the tests
	tree := NewTree(WithGridSize(0.01), WithMaxDepth(3), WithMinPointsPerChildren(1), WithMaxPointsPerTile(5))
	reader := &las.MockLasReader{
		CRS: "EPSG:4978",
		Pts: []geom.Point64{
			{Vector: model.Vector{X: 0, Y: 0, Z: 0}},
			{Vector: model.Vector{X: 0.1, Y: 0.1, Z: 0.1}},
			{Vector: model.Vector{X: 0.2, Y: 0.2, Z: 0.2}},
			{Vector: model.Vector{X: 0.3, Y: 0.3, Z: 0.3}},
			{Vector: model.Vector{X: 0.4, Y: 0.4, Z: 0.4}},
			{Vector: model.Vector{X: 0.5, Y: 0.5, Z: 0.5}},
			{Vector: model.Vector{X: 0.6, Y: 0.6, Z: 0.6}},
			{Vector: model.Vector{X: 0.7, Y: 0.7, Z: 0.7}},
			{Vector: model.Vector{X: 0.8, Y: 0.8, Z: 0.8}},
			{Vector: model.Vector{X: 0.9, Y: 0.9, Z: 0.9}},
		},
	}
	conv := test.GetTestCoordinateConverterFactory()
	tree.Load(reader, conv, nil, context.TODO())
	err := tree.Build()
	if err != nil {
		t.Fatalf("unexpected error during tree build: %v", err)
	}
	root := tree.RootNode()
	if actual := root.NumberOfPoints(); actual != 5 {
		t.Errorf("expected 5 points, got %d", actual)
	}
	if actual := root.TotalNumberOfPoints(); actual != 10 {
		t.Errorf("expected 10 points, got %d", actual)
	}
}
