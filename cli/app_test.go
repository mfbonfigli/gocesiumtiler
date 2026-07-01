package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mfbonfigli/gotiler-core/tiler"
	"github.com/mfbonfigli/gotiler-core/tiler/model"
	"github.com/mfbonfigli/gotiler-core/tiler/mutator"
	"github.com/mfbonfigli/gotiler-core/tiler/plugin"
	urfavecli "github.com/urfave/cli/v2"
)

func TestDefaultTiler(t *testing.T) {
	tl, err := DefaultTilerProvider()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	switch tl.(type) {
	case *tiler.GoTiler:
	default:
		t.Errorf("unexpected tiler type returned")
	}
}

func TestNewAppSupportsBrandingAndExtraCommands(t *testing.T) {
	called := false
	app := NewApp(Options{
		BuildInfo: BuildInfo{Version: "9.9.9", GitCommit: "abc"},
		Branding: Branding{
			Name:  "gotiler-cli",
			Usage: "private tiler",
		},
		ExtraCommands: []CommandFactory{
			func(ctx Context) *urfavecli.Command {
				if ctx.BuildInfo.VersionString() != "9.9.9-abc" {
					t.Fatalf("unexpected build info in command context: %s", ctx.BuildInfo.VersionString())
				}
				return &urfavecli.Command{
					Name: "extra",
					Action: func(*urfavecli.Context) error {
						called = true
						return nil
					},
				}
			},
		},
	})
	if app.Name != "gotiler-cli" || app.Usage != "private tiler" || app.Version != "9.9.9-abc" {
		t.Fatalf("unexpected app metadata: name=%q usage=%q version=%q", app.Name, app.Usage, app.Version)
	}
	if app.Command("version") == nil {
		t.Fatal("expected version command to be registered")
	}
	if app.Command("file") != nil || app.Command("folder") != nil || app.Command("pointcloud") != nil || app.Command("pc") != nil || app.Command("pcloud") != nil {
		t.Fatal("expected point-cloud commands to be absent by default")
	}
	if err := app.Run([]string{"gotiler-cli", "extra"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected extra command to be called")
	}
}

func TestMainVersionCommand(t *testing.T) {
	called := false
	app := NewApp(Options{
		BuildInfo: BuildInfo{Version: "9.9.9", GitCommit: "abc"},
		TilerProvider: func() (tiler.Tiler, error) {
			called = true
			return &tiler.MockTiler{}, nil
		},
	})
	buf := &bytes.Buffer{}
	app.Writer = buf

	if err := app.Run([]string{"gotiler", "version"}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected version command to not initialize the tiler")
	}
	if actual := buf.String(); actual != "9.9.9-abc\n" {
		t.Fatalf("expected version output %q, got %q", "9.9.9-abc\n", actual)
	}
}

func TestMainWithoutArgsShowsHelp(t *testing.T) {
	called := false
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		called = true
		return &tiler.MockTiler{}, nil
	}})
	buf := &bytes.Buffer{}
	app.Writer = buf

	err := app.Run([]string{"gotiler"})
	if err == nil {
		t.Fatal("expected missing input to fail")
	}
	if called {
		t.Fatal("expected missing input to not initialize the tiler")
	}
	out := buf.String()
	if !strings.Contains(out, "COMMANDS") || !strings.Contains(out, "version") || !strings.Contains(out, "--out") {
		t.Fatalf("expected app help to include commands, version, and point-cloud flags, got %q", out)
	}
}

func TestConfiguredPointCloudCommandWithoutArgsShowsHelp(t *testing.T) {
	called := false
	app := NewApp(Options{
		PointCloudCommandName: "pcloud",
		TilerProvider: func() (tiler.Tiler, error) {
			called = true
			return &tiler.MockTiler{}, nil
		},
	})
	buf := &bytes.Buffer{}
	app.Writer = buf

	err := app.Run([]string{"gotiler", "pcloud"})
	if err == nil {
		t.Fatal("expected missing input to fail")
	}
	if called {
		t.Fatal("expected missing input to not initialize the tiler")
	}
	out := buf.String()
	if !strings.Contains(out, "pcloud") || !strings.Contains(out, "--out") {
		t.Fatalf("expected pcloud help to include command name and flags, got %q", out)
	}
}

func TestMainProcessFile(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "myfile.las")
	if err := touchFile(input); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		return mockTiler, nil
	}})
	err := app.Run([]string{"gotiler",
		"-out", ".\\abc",
		"-crs", "EPSG:4979",
		"-z-offset", "-1",
		"-points-per-tile", "5000",
		"-8-bit",
		"-r", "replace",
		"--initial-geometric-error", "128",
		"--ge-correction", "1.5",
		"--attributes", "intensity",
		input})
	if err != nil {
		t.Fatal(err)
	}
	if mockTiler.ProcessFilesCalled != true {
		t.Error("expected processFiles called but was not")
	}
	if actual := mockTiler.InputFiles; !reflect.DeepEqual(actual, []string{input}) {
		t.Errorf("expected tiler to be called with %v but got %v", []string{input}, actual)
	}
	if actual := mockTiler.SourceCRS; actual != "EPSG:4979" {
		t.Errorf("expected tiler to be called with epsg %v but got epsg %v", 4979, actual)
	}
	if actual := mockTiler.OutputFolder; actual != ".\\abc" {
		t.Errorf("expected tiler to be called with output folder %v but got %v", ".\\abc", actual)
	}
	if actual := mockTiler.EightBit; actual != true {
		t.Errorf("expected tiler to be called with EightBit %v but got %v", true, actual)
	}
	if actual := mockTiler.PtsPerTile; actual != 5000 {
		t.Errorf("expected tiler to be called with PtsPerTile %v but got %v", 5000, actual)
	}
	if actual := mockTiler.Mutators[0].(*mutator.ZOffset).Offset; actual != -1 {
		t.Errorf("expected tiler to be called with ZOffset mutator with offset %v but got %v", -1, actual)
	}
	if actual := len(mockTiler.Mutators); actual != 1 {
		t.Errorf("expected 1 mutator but got %v", actual)
	}
	if actual := mockTiler.RefineMode; actual != model.RefineReplace {
		t.Errorf("expected tiler to be called with refine mode %v but got %v", "replace", actual)
	}
	if actual := mockTiler.EncoderID; actual != plugin.EncoderGLB {
		t.Errorf("expected tiler to be called with encoder %v but got %v", plugin.EncoderGLB, actual)
	}
	if actual := mockTiler.InitialGeometricError; actual != 128 {
		t.Errorf("expected tiler to be called with InitialGeometricError %v but got %v", 128, actual)
	}
	if actual := mockTiler.GECorrection; actual != 1.5 {
		t.Errorf("expected tiler to be called with GECorrection %v but got %v", 1.5, actual)
	}
	if !mockTiler.Attributes.Has(model.AttrIntensity) || mockTiler.Attributes.Has(model.AttrClassification) {
		t.Errorf("expected Attributes to contain only intensity, got %v", mockTiler.Attributes)
	}
}

func TestMainProcessFolder(t *testing.T) {
	input := t.TempDir()
	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		return mockTiler, nil
	}})
	err := app.Run([]string{"gotiler",
		"-out", ".\\abc",
		"-c", "4979",
		"-z-offset", "-1",
		"-points-per-tile", "5000",
		"-8-bit",
		"-v", "1.0",
		"-refine-mode", "add",
		input})
	if err != nil {
		t.Fatal(err)
	}
	if mockTiler.ProcessFolderCalled != true {
		t.Error("expected processFolder called but was not")
	}
	if actual := mockTiler.InputFolder; !reflect.DeepEqual(actual, input) {
		t.Errorf("expected tiler to be called with %v but got %v", input, actual)
	}
	if actual := mockTiler.SourceCRS; actual != "EPSG:4979" {
		t.Errorf("expected tiler to be called with epsg %v but got epsg %v", 4979, actual)
	}
	if actual := mockTiler.OutputFolder; actual != ".\\abc" {
		t.Errorf("expected tiler to be called with output folder %v but got %v", ".\\abc", actual)
	}
	if actual := mockTiler.EightBit; actual != true {
		t.Errorf("expected tiler to be called with EightBit %v but got %v", true, actual)
	}
	if actual := mockTiler.PtsPerTile; actual != 5000 {
		t.Errorf("expected tiler to be called with PtsPerTile %v but got %v", 5000, actual)
	}
	if actual := mockTiler.Mutators[0].(*mutator.ZOffset).Offset; actual != -1 {
		t.Errorf("expected tiler to be called with ZOffset mutator with offset %v but got %v", -1, actual)
	}
	if actual := len(mockTiler.Mutators); actual != 1 {
		t.Errorf("expected 1 mutator but got %v", actual)
	}
	if actual := mockTiler.RefineMode; actual != model.RefineAdd {
		t.Errorf("expected tiler to be called with refine mode %v but got %v", "replace", actual)
	}
	if actual := mockTiler.EncoderID; actual != plugin.EncoderPNTS {
		t.Errorf("expected tiler to be called with encoder %v but got %v", plugin.EncoderPNTS, actual)
	}
	if actual := mockTiler.InitialGeometricError; actual != 0 {
		t.Errorf("expected default InitialGeometricError to be 0 (auto) but got %v", actual)
	}
}

func TestConfiguredPointCloudCommandFolderJoin(t *testing.T) {
	tmp, err := os.MkdirTemp(os.TempDir(), "tst")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	touchFile(filepath.Join(tmp, "test0.las"))
	touchFile(filepath.Join(tmp, "test0.xyz"))
	touchFile(filepath.Join(tmp, "test1.LAS"))
	touchFile(filepath.Join(tmp, "test2.LAS"))

	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{
		PointCloudCommandName: "pcloud",
		TilerProvider: func() (tiler.Tiler, error) {
			return mockTiler, nil
		},
	})
	err = app.Run([]string{"gotiler", "pcloud",
		"-out", ".\\abc",
		"-crs", "4979",
		"-z-offset", "-1",
		"-p", "5000",
		"-8-bit",
		"-v", "1.1",
		"-join",
		tmp})
	if err != nil {
		t.Fatal(err)
	}
	if mockTiler.ProcessFolderCalled != false {
		t.Error("expected processFolder to not be called but it was")
	}
	if mockTiler.ProcessFilesCalled != true {
		t.Error("expected processFiles called but was not")
	}
	expected := []string{
		filepath.Join(tmp, "test0.las"),
		filepath.Join(tmp, "test1.LAS"),
		filepath.Join(tmp, "test2.LAS"),
	}
	if actual := mockTiler.InputFiles; !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected tiler to be called with %v but got %v", expected, actual)
	}
	if actual := mockTiler.SourceCRS; actual != "EPSG:4979" {
		t.Errorf("expected tiler to be called with epsg %v but got epsg %v", 4979, actual)
	}
	if actual := mockTiler.OutputFolder; actual != ".\\abc" {
		t.Errorf("expected tiler to be called with output folder %v but got %v", ".\\abc", actual)
	}
	if actual := mockTiler.EightBit; actual != true {
		t.Errorf("expected tiler to be called with EightBit %v but got %v", true, actual)
	}
	if actual := mockTiler.PtsPerTile; actual != 5000 {
		t.Errorf("expected tiler to be called with PtsPerTile %v but got %v", 5000, actual)
	}
	if actual := mockTiler.Mutators[0].(*mutator.ZOffset).Offset; actual != -1 {
		t.Errorf("expected tiler to be called with ZOffset mutator with offset %v but got %v", -1, actual)
	}
	if actual := len(mockTiler.Mutators); actual != 1 {
		t.Errorf("expected 1 mutator but got %v", actual)
	}
	if actual := mockTiler.EncoderID; actual != plugin.EncoderGLB {
		t.Errorf("expected tiler to be called with encoder %v but got %v", plugin.EncoderGLB, actual)
	}
}

func TestMainProcessesFileWithoutSubcommand(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "myfile.las")
	if err := touchFile(input); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		return mockTiler, nil
	}})
	err := app.Run([]string{"gotiler",
		"-out", ".\\abc",
		"-crs", "4979",
		"-points-per-tile", "5000",
		input})
	if err != nil {
		t.Fatal(err)
	}
	if !mockTiler.ProcessFilesCalled {
		t.Fatal("expected processFiles called but was not")
	}
	if actual := mockTiler.InputFiles; !reflect.DeepEqual(actual, []string{input}) {
		t.Errorf("expected tiler to be called with %v but got %v", []string{input}, actual)
	}
	if actual := mockTiler.SourceCRS; actual != "EPSG:4979" {
		t.Errorf("expected tiler to be called with epsg %v but got epsg %v", 4979, actual)
	}
	if actual := mockTiler.OutputFolder; actual != ".\\abc" {
		t.Errorf("expected tiler to be called with output folder %v but got %v", ".\\abc", actual)
	}
}

func TestMainProcessesFolderJoinWithoutSubcommand(t *testing.T) {
	tmp := t.TempDir()
	touchFile(filepath.Join(tmp, "test0.las"))
	touchFile(filepath.Join(tmp, "test1.LAS"))
	touchFile(filepath.Join(tmp, "test2.xyz"))

	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		return mockTiler, nil
	}})
	err := app.Run([]string{"gotiler",
		"-out", ".\\abc",
		"-crs", "4979",
		"-points-per-tile", "5000",
		"-join",
		tmp})
	if err != nil {
		t.Fatal(err)
	}
	if mockTiler.ProcessFolderCalled {
		t.Fatal("expected processFolder to not be called")
	}
	if !mockTiler.ProcessFilesCalled {
		t.Fatal("expected processFiles called but was not")
	}
	expected := []string{
		filepath.Join(tmp, "test0.las"),
		filepath.Join(tmp, "test1.LAS"),
	}
	if actual := mockTiler.InputFiles; !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected tiler to be called with %v but got %v", expected, actual)
	}
}

func TestMainProcessFileRejectsJoin(t *testing.T) {
	tmp := t.TempDir()
	input := filepath.Join(tmp, "myfile.las")
	if err := touchFile(input); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	mockTiler := &tiler.MockTiler{}
	app := NewApp(Options{TilerProvider: func() (tiler.Tiler, error) {
		return mockTiler, nil
	}})
	err := app.Run([]string{"gotiler", "-out", ".\\abc", "--join", input})
	if err == nil {
		t.Fatal("expected --join with file input to fail")
	}
	if mockTiler.ProcessFilesCalled || mockTiler.ProcessFolderCalled {
		t.Fatal("expected no tiler processing after invalid file --join input")
	}
}
