package unit

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters/elevation/offset_elevation_corrector"
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"github.com/mfbonfigli/gocesiumtiler/pkg/algorithm_manager/std_algorithm_manager"
	"reflect"
	"testing"
)

func TestAlgorithmManagerReturnsGridTree(t *testing.T) {
	expected := "GridTree"
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm: tiler.Grid,
		},
	)

	treeType := reflect.ValueOf(algorithmManager.GetTreeAlgorithm()).Elem().Type().Name()
	if treeType != expected {
		t.Errorf("Wrong tree algorithm returned, %s expected, but %s was returned", expected, treeType)
	}
}

func TestAlgorithmManagerReturnsRandomTree(t *testing.T) {
	expectedTree := "RandomTree"
	expectedLoader := "RandomLoader"
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm: tiler.Random,
		},
	)

	value := reflect.ValueOf(algorithmManager.GetTreeAlgorithm())
	treeType := value.Elem().Type().Name()
	if treeType != expectedTree {
		t.Errorf("Wrong tree algorithm returned, %s expectedTree, but %s was returned", expectedTree, treeType)
	}

	loaderType := value.Elem().FieldByName("Loader").Elem().Elem().Type().Name()

	if loaderType != expectedLoader {
		t.Errorf("Wrong tree loader returned, %s expected but %s was returned", expectedLoader, loaderType)
	}
}

func TestAlgorithmManagerReturnsRandomBoxTree(t *testing.T) {
	expectedTree := "RandomTree"
	expectedLoader := "RandomBoxLoader"
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm: tiler.RandomBox,
		},
	)

	value := reflect.ValueOf(algorithmManager.GetTreeAlgorithm())
	treeType := value.Elem().Type().Name()
	if treeType != expectedTree {
		t.Errorf("Wrong tree algorithm returned, %s expectedTree, but %s was returned", expectedTree, treeType)
	}

	loaderType := value.Elem().FieldByName("Loader").Elem().Elem().Type().Name()

	if loaderType != expectedLoader {
		t.Errorf("Wrong tree loader returned, %s expected but %s was returned", expectedLoader, loaderType)
	}
}

func TestAlgorithmManagerReturnsProj4CoordinateConverter(t *testing.T) {
	expected := "proj4CoordinateConverter"
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm: tiler.Grid,
		},
	)

	coordinateConverterType := reflect.ValueOf(algorithmManager.GetCoordinateConverterAlgorithm()).Elem().Type().Name()
	if coordinateConverterType != expected {
		t.Errorf("Wrong coordinate converter algorithm returned, %s expected, but %s was returned", expected, coordinateConverterType)
	}
}

func TestAlgorithmManagerReturnsOffsetElevationCorrector(t *testing.T) {
	expectedWrapper := "PipelineElevationCorrector"
	expectedNestedCorrector := "OffsetElevationCorrector"
	expectedOffset := 10.3
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm:              tiler.Grid,
			ZOffset:                expectedOffset,
			EnableGeoidZCorrection: false,
		},
	)

	elevationCorrectionType := reflect.ValueOf(algorithmManager.GetElevationCorrectionAlgorithm()).Elem()
	treeType := elevationCorrectionType.Type().Name()
	if treeType != expectedWrapper {
		t.Fatalf("Wrong elevation correction algorithm returned, %s expected, but %s was returned", expectedWrapper, treeType)
	}

	correctors := elevationCorrectionType.FieldByName("Correctors").Interface().([]converters.ElevationCorrector)

	if len(correctors) != 1 {
		t.Fatalf("One nested correction algorithm expected but %d found", len(correctors))
	}

	nestedCorrector := reflect.ValueOf(correctors[0]).Elem().Type().Name()
	if nestedCorrector != expectedNestedCorrector {
		t.Fatalf("Wrong elevation corrector algorithm returned, %s expected, but %s was returned", expectedNestedCorrector, nestedCorrector)
	}

	actualOffset := correctors[0].(*offset_elevation_corrector.OffsetElevationCorrector).Offset
	if expectedOffset != actualOffset {
		t.Errorf("Expected offset %f but got %f", expectedOffset, actualOffset)
	}
}

func TestAlgorithmManagerReturnsGeoidElevationCorrector(t *testing.T) {
	expectedWrapper := "PipelineElevationCorrector"
	expectedNestedCorrectorOne := "OffsetElevationCorrector"
	expectedNestedCorrectorTwo := "GeoidElevationCorrector"
	expectedOffset := 10.3
	algorithmManager := std_algorithm_manager.NewAlgorithmManager(
		&tiler.TilerOptions{
			Algorithm:              tiler.Grid,
			ZOffset:                expectedOffset,
			EnableGeoidZCorrection: true,
		},
	)

	elevationCorrectionType := reflect.ValueOf(algorithmManager.GetElevationCorrectionAlgorithm()).Elem()
	treeType := elevationCorrectionType.Type().Name()
	if treeType != expectedWrapper {
		t.Fatalf("Wrong elevation correction algorithm returned, %s expected, but %s was returned", expectedWrapper, treeType)
	}

	correctors := elevationCorrectionType.FieldByName("Correctors").Interface().([]converters.ElevationCorrector)

	if len(correctors) != 2 {
		t.Fatalf("Two nested correction algorithms expected but %d found", len(correctors))
	}

	nestedCorrector := reflect.ValueOf(correctors[0]).Elem().Type().Name()
	if nestedCorrector != expectedNestedCorrectorOne {
		t.Fatalf("Wrong first elevation corrector algorithm returned, %s expected, but %s was returned", expectedNestedCorrectorOne, nestedCorrector)
	}

	actualOffset := correctors[0].(*offset_elevation_corrector.OffsetElevationCorrector).Offset
	if expectedOffset != actualOffset {
		t.Errorf("Expected offset %f but got %f", expectedOffset, actualOffset)
	}

	nestedCorrector = reflect.ValueOf(correctors[1]).Elem().Type().Name()
	if nestedCorrector != expectedNestedCorrectorTwo {
		t.Fatalf("Wrong second elevation corrector algorithm returned, %s expected, but %s was returned", expectedNestedCorrectorOne, nestedCorrector)
	}

}
