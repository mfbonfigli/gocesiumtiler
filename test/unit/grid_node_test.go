package unit

import (
	"math"
	"testing"

	"github.com/mfbonfigli/gocesiumtiler/internal/data"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree/grid_tree"
)

func TestGridNodeAddDataPointSinglePoint(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		5.0,
		1.0,
	)

	point := data.NewPoint(14, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	if len(node.GetPoints()) != 1 {
		t.Fatalf("One point expected, %d returned", len(node.GetPoints()))
	}

	if node.GetPoints()[0] != point {
		t.Errorf("Unexpected point data returned")
	}

	if node.NumberOfPoints() != 1 {
		t.Fatalf("Expected NumberOfPoints %d, got %d returned", 1, node.NumberOfPoints())
	}

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeAddDataPointMultiplePoints(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		10.0,
		1.0,
	)

	point := data.NewPoint(11, 11, 1, 2, 3, 4, 5, 6)
	point2 := data.NewPoint(13, 13, 1, 2, 3, 4, 5, 6)
	point3 := data.NewPoint(12, 12, 1, 2, 3, 4, 5, 6)

	node.AddDataPoint(point)
	node.AddDataPoint(point2)
	node.AddDataPoint(point3)

	node.(*grid_tree.GridNode).BuildPoints()

	if len(node.GetPoints()) != 1 {
		t.Fatalf("One point expected, %d returned", len(node.GetPoints()))
	}

	if node.GetPoints()[0] != point2 {
		t.Errorf("Unexpected point data returned")
	}

	if node.NumberOfPoints() != 1 {
		t.Errorf("Expected NumberOfPoints %d, got %d returned", 1, node.NumberOfPoints())
	}

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeGetInternalSrid(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		5.0,
		1.0,
	)

	if node.GetInternalSrid() != 3395 {
		t.Errorf("Expected Internal Srid %d, got %d returned", 3395, node.GetInternalSrid())
	}

	if node.IsEmpty() != true {
		t.Fatalf("Expected IsEmpty %v, got %v returned", true, node.IsEmpty())
	}
}

func TestGridNodeGetIsRootTrue(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		5.0,
		1.0,
	)

	if node.IsRoot() != true {
		t.Errorf("Expected IsRoot %t, got %t", true, node.IsRoot())
	}

	if node.IsEmpty() != true {
		t.Fatalf("Expected IsEmpty %v, got %v returned", true, node.IsEmpty())
	}
}

func TestGridNodeGetIsRootFalse(t *testing.T) {
	node := grid_tree.NewGridNode(
		grid_tree.NewGridNode(nil, geometry.NewBoundingBox(14, 15, 41, 42, 1, 2), 5.0, 1.0),
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		5.0,
		1.0,
	)

	if node.IsRoot() != false {
		t.Errorf("Expected IsRoot %t, got %t", false, node.IsRoot())
	}

	if node.IsEmpty() != true {
		t.Fatalf("Expected IsEmpty %v, got %v returned", true, node.IsEmpty())
	}
}

func TestGridNodeGetBoundingBoxRegion(t *testing.T) {
	inputRegion := geometry.NewBoundingBox(14, 15, 41, 42, 1, 2)
	node := grid_tree.NewGridNode(
		nil,
		inputRegion,
		5.0,
		1.0,
	)

	region, _ := node.GetBoundingBoxRegion(&mockCoordinateConverter{}, 0, 0, 0)

	if region != inputRegion {
		t.Errorf("Expected region equal to node bounding box")
	}

	if node.IsEmpty() != true {
		t.Fatalf("Expected IsEmpty %v, got %v returned", true, node.IsEmpty())
	}
}

func TestGridNodeGetChildren(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		5.0,
		1.0,
	)

	point := data.NewPoint(14, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)
	point = data.NewPoint(15, 42, 2, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)
	point = data.NewPoint(15, 42, 2, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	children := node.GetChildren()
	if len(children[0].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[0].GetPoints()))
	}
	if len(children[1].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[1].GetPoints()))
	}
	if len(children[2].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[2].GetPoints()))
	}
	if len(children[3].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[3].GetPoints()))
	}
	if len(children[4].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[4].GetPoints()))
	}
	if len(children[5].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[5].GetPoints()))
	}
	if len(children[6].GetPoints()) != 0 {
		t.Errorf("Expected children 1 to have %d points but got %d", 0, len(children[6].GetPoints()))
	}
	if len(children[7].GetPoints()) != 1 {
		t.Errorf("Expected children 1 to have %d points but got %d", 1, len(children[7].GetPoints()))
	}
}

func TestGridNodeGetPoints(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		1.0,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.3, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.2, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	children := node.GetChildren()

	if len(node.GetPoints()) != 1 {
		t.Errorf("Expected node to have %d points but got %d", 1, len(node.GetPoints()))
	}
	if node.GetPoints()[0].X != 14.3 {
		t.Errorf("Expected point in node to have %f X coordinate but got %f", 14.3, node.GetPoints()[0].X)
	}
	if len(children[0].GetPoints()) != 2 {
		t.Errorf("Expected children 1 to have %d points but got %d", 2, len(children[0].GetPoints()))
	}

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeGetTotalNumberOfPoints(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		1.0,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.3, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.2, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeGetNumberOfPoints(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.3, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.2, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	if node.NumberOfPoints() != 1 {
		t.Errorf("Expected node to have NumberOfPoints equal to %d but got %d", 1, node.NumberOfPoints())
	}

	if node.NumberOfPoints() != int32(len(node.GetPoints())) {
		t.Errorf("Expected node to have NumberOfPoints equal to length of GetPoints array %d but got %d", len(node.GetPoints()), node.NumberOfPoints())
	}

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeIsLeaf(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.3, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	if node.IsLeaf() {
		t.Errorf("Expected node to be non leaf")
	}

	if !node.GetChildren()[0].IsLeaf() {
		t.Errorf("Expected children 0 to be a leaf")
	}

	if node.IsEmpty() != false {
		t.Fatalf("Expected IsEmpty %v, got %v returned", false, node.IsEmpty())
	}
}

func TestGridNodeIsInitialized(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	if !node.IsInitialized() {
		t.Errorf("Expected node to be initialized")
	}
}

func TestGridNodeComputeGeometricError(t *testing.T) {
	node := grid_tree.NewGridNode(
		&grid_tree.GridNode{},
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	expectedError := 1.0 * math.Sqrt(3) * 2
	if node.ComputeGeometricError(0, 0, 0) != expectedError {
		t.Errorf("Expected ComputeGeometricError %f, got %f", expectedError, node.ComputeGeometricError(0, 0, 0))
	}
}

func TestRootGridNodeComputeGeometricError(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 16, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	expectedError := 1.0 * math.Sqrt(4+1+1)
	if node.ComputeGeometricError(0, 0, 0) != expectedError {
		t.Errorf("Expected ComputeGeometricError %f, got %f", expectedError, node.ComputeGeometricError(0, 0, 0))
	}
}

func TestGridNodeGetParent(t *testing.T) {
	node := grid_tree.NewGridNode(
		nil,
		geometry.NewBoundingBox(14, 15, 41, 42, 1, 2),
		1.0,
		0.5,
	)

	point := data.NewPoint(14.1, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	point = data.NewPoint(14.3, 41, 1, 2, 3, 4, 5, 6)
	node.AddDataPoint(point)

	node.(*grid_tree.GridNode).BuildPoints()

	if node.GetParent() != nil {
		t.Errorf("Unexpected parent node")
	}

	if node.GetChildren()[0].GetParent() != node {
		t.Errorf("Unexpected parent node")
	}
}
