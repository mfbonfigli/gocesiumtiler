package test

import (
	"flag"
	"github.com/mfbonfigli/gocesiumtiler/utils"
	"os"
	"strconv"
	"testing"
)

func TestInputFlagIsParsed(t *testing.T) {
	expected := "/home/user/file.las"
	os.Args = []string{"gocesiumtiler", "-input=" + expected}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Input != expected {
		t.Errorf("Expected Input = %s, got %s", expected, *flags.Input)
	}
}

func TestOutputFlagIsParsed(t *testing.T) {
	expected := "/home/user/output"
	os.Args = []string{"gocesiumtiler", "-output=" + expected}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Output != expected {
		t.Errorf("Expected Output = %s, got %s", expected, *flags.Output)
	}
}

func TestSridFlagIsParsed(t *testing.T) {
	expected := 32633
	os.Args = []string{"gocesiumtiler", "-srid=" + strconv.Itoa(expected)}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Srid != expected {
		t.Errorf("Expected Srid = %d, got %d", expected, *flags.Srid)
	}
}

func TestSridFlagDefaultIs4326(t *testing.T) {
	expected := 4326
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Srid != expected {
		t.Errorf("Expected Srid = %d, got %d", expected, *flags.Srid)
	}
}

func TestZOffsetFlagIsParsed(t *testing.T) {
	expected := 10.0
	os.Args = []string{"gocesiumtiler", "-zoffset=10"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.ZOffset != expected {
		t.Errorf("Expected ZOffset = %f, got %f", expected, *flags.ZOffset)
	}
}

func TestZOffsetFlagDefaultIsZero(t *testing.T) {
	expected := 0.0
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.ZOffset != expected {
		t.Errorf("Expected ZOffset = %f, got %f", expected, *flags.ZOffset)
	}
}

func TestMaxPtsFlagIsParsed(t *testing.T) {
	expected := 2000
	os.Args = []string{"gocesiumtiler", "-maxpts=2000"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.MaxNumPts != expected {
		t.Errorf("Expected MaxNumPts = %d, got %d", expected, *flags.MaxNumPts)
	}
}

func TestMaxPtsFlagDefaultIs50000(t *testing.T) {
	expected := 50000
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.MaxNumPts != expected {
		t.Errorf("Expected MaxNumPts = %d, got %d", expected, *flags.MaxNumPts)
	}
}

func TestGeoidFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-geoid"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.ZGeoidCorrection {
		t.Errorf("Expected ZGeoidCorrection = %t, got %t", expected, *flags.ZGeoidCorrection)
	}
}

func TestGeoidFlagDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.ZGeoidCorrection {
		t.Errorf("Expected ZGeoidCorrection = %t, got %t", expected, *flags.ZGeoidCorrection)
	}
}

func TestFolderProcessingFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-folder"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.FolderProcessing {
		t.Errorf("Expected FolderProcessing = %t, got %t", expected, *flags.FolderProcessing)
	}
}

func TestFolderProcessingDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.FolderProcessing {
		t.Errorf("Expected FolderProcessing = %t, got %t", expected, *flags.FolderProcessing)
	}
}

func TestRecursiveFolderProcessingFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-recursive"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.RecursiveFolderProcessing {
		t.Errorf("Expected RecursiveFolderProcessing = %t, got %t", expected, *flags.RecursiveFolderProcessing)
	}
}

func TestRecursiveFolderProcessingDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.RecursiveFolderProcessing {
		t.Errorf("Expected RecursiveFolderProcessing = %t, got %t", expected, *flags.RecursiveFolderProcessing)
	}
}

func TestSilentFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-silent"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.Silent {
		t.Errorf("Expected Silent = %t, got %t", expected, *flags.Silent)
	}
}

func TestSilentFlagDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Silent {
		t.Errorf("Expected Silent = %t, got %t", expected, *flags.Silent)
	}
}

func TestLogTimestampFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-timestamp"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.LogTimestamp {
		t.Errorf("Expected LogTimestamp = %t, got %t", expected, *flags.LogTimestamp)
	}
}

func TestLogTimestampFlagDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.LogTimestamp {
		t.Errorf("Expected LogTimestamp = %t, got %t", expected, *flags.LogTimestamp)
	}
}

func TestHqFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-hq"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.Hq {
		t.Errorf("Expected Hq = %t, got %t", expected, *flags.Hq)
	}
}

func TestHqDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Hq {
		t.Errorf("Expected Hq = %t, got %t", expected, *flags.Hq)
	}
}

func TestHelpFlagIsParsed(t *testing.T) {
	expected := true
	os.Args = []string{"gocesiumtiler", "-help"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if !*flags.Help {
		t.Errorf("Expected Help = %t, got %t", expected, *flags.Help)
	}
}

func TestHelpDefaultIsFalse(t *testing.T) {
	expected := false
	os.Args = []string{"gocesiumtiler"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags := utils.ParseFlags()
	if *flags.Help {
		t.Errorf("Expected Help = %t, got %t", expected, *flags.Help)
	}
}

