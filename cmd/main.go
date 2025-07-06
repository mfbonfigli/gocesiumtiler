package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler"
	"github.com/mfbonfigli/gocesiumtiler/v2/tiler/mutator"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
	"github.com/urfave/cli/v2"
)

// this global variable controls the tiler that will be used. Useful to inject mocks during tests.
var tilerProvider func() (tiler.Tiler, error) = func() (tiler.Tiler, error) {
	return tiler.NewGoCesiumTiler()
}

var cmdVersion = "2.0.1"

// GitCommit is injected dynamically at build time via `go build -ldflags "-X main.GitCommit=XYZ"`
var GitCommit string = "(na)"

const logo = `
                           _                 _   _ _
  __ _  ___   ___ ___  ___(_)_   _ _ __ ___ | |_(_) | ___ _ __ 
 / _  |/ _ \ / __/ _ \/ __| | | | | '_   _ \| __| | |/ _ \ '__|
| (_| | (_) | (_|  __/\__ \ | |_| | | | | | | |_| | |  __/ |   
 \__, |\___/ \___\___||___/_|\__,_|_| |_| |_|\__|_|_|\___|_|   
  __| | A Cesium Point Cloud tile generator written in golang
 |___/  Copyright YYYY - Massimo Federico Bonfigli    
        build: ZZZZ

 `

var profilerEnabled = false

func main() {
	printBanner()
	getCli(defaultCliOptions()).Run(os.Args)
}

func getVersion() string {
	return fmt.Sprintf("%s-%s", cmdVersion, GitCommit)
}

func getCli(c *cliOpts) *cli.App {
	return &cli.App{
		Name:    "gocesiumtiler",
		Usage:   "transforms LAS files into Cesium.JS 3D Tiles",
		Version: getVersion(),
		Commands: []*cli.Command{
			{
				Name:  "file",
				Usage: "convert a LAS file into 3D tiles",
				Flags: getFileFlags(c),
				Action: func(cCtx *cli.Context) error {
					fileCommand(c, cCtx.Args().First())
					return nil
				},
			},
			{
				Name:  "folder",
				Usage: "convert all LAS files in a folder file into 3D tiles",
				Flags: getFolderFlags(c),
				Action: func(cCtx *cli.Context) error {
					folderCommand(c, cCtx.Args().First())
					return nil
				},
			},
		},
		EnableBashCompletion: true,
	}
}

func getFileFlags(c *cliOpts) []cli.Flag {
	return getFlags(c)
}

func getFolderFlags(c *cliOpts) []cli.Flag {
	stdFlags := getFlags(c)
	joinFlag := &cli.BoolFlag{
		Name:        "join",
		Aliases:     []string{"j"},
		Value:       c.join,
		Usage:       "merge the input LAS files in the folder into a single cloud. The LAS files must have the same properties (CRS etc)",
		Destination: &c.join,
	}
	return append(stdFlags, joinFlag)
}

func getFlags(c *cliOpts) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "out",
			Aliases:     []string{"o"},
			Value:       c.output,
			Usage:       "full path of the output folder where to save the resulting Cesium tilesets",
			Destination: &c.output,
		},
		&cli.StringFlag{
			Name:        "crs",
			Aliases:     []string{"e", "epsg"},
			Value:       c.crs,
			Usage:       "String representing the input CRS. For example, EPSG:4326 or a generic Proj4 string. Bare numbers will be interpreted as EPSG codes. If empty the system will attempt to autodetect the CRS from the LAS metadata. In case of multiple LAS files, the CRS must be consistent else an error will be thrown.",
			Destination: &c.crs,
		},
		&cli.Float64Flag{
			Name:        "resolution",
			Aliases:     []string{"r"},
			Value:       c.resolution,
			Usage:       "minimum resolution of the 3d tiles, in meters. approximately represets the maximum sampling distance between any two points at the lowest level of detail",
			Destination: &c.resolution,
		},
		&cli.Float64Flag{
			Name:        "z-offset",
			Aliases:     []string{"z"},
			Value:       c.zOffset,
			Usage:       "z offset to apply to the point, in meters. only use it if the input elevation is referred to the WGS84 ellipsoid or geoid",
			Destination: &c.zOffset,
		},
		&cli.IntFlag{
			Name:        "depth",
			Aliases:     []string{"d"},
			Value:       c.maxDepth,
			Usage:       "maximum depth of the output tree.",
			Destination: &c.maxDepth,
		},
		&cli.IntFlag{
			Name:        "min-points-per-tile",
			Aliases:     []string{"m"},
			Value:       c.minPoints,
			Usage:       "minimum number of points to enforce in each 3D tile. Might be ignored when the parent has not enough capacity to store the child points (see max-points-per-tile)",
			Destination: &c.minPoints,
		},
		&cli.IntFlag{
			Name:        "max-points-per-tile",
			Aliases:     []string{"M"},
			Value:       c.maxPoints,
			Usage:       "maximum number of points a tile can contain. If a tile contains more points, a subsampling strategy is applied. 0 means no limit.",
			Destination: &c.maxPoints,
		},
		&cli.BoolFlag{
			Name:        "8-bit",
			Value:       c.eightBit,
			Usage:       "set to interpret the input points color as part of a 8bit color space",
			Destination: &c.eightBit,
		},
		&cli.Float64Flag{
			Name:        "subsample",
			Value:       c.subsamplePct,
			Usage:       "Approximate percent of points to keep in the final point cloud, between 0.01 (1%) and 1 (100%)",
			Destination: &c.subsamplePct,
		},
		&cli.StringFlag{
			Name:        "version",
			Aliases:     []string{"v"},
			Value:       c.version,
			Usage:       "sets the version of the tileset to generate. Could be either 1.0 or 1.1",
			Destination: &c.version,
		},
	}
}

type cliOpts struct {
	output       string
	crs          string
	maxDepth     int
	minPoints    int
	maxPoints    int
	resolution   float64
	zOffset      float64
	subsamplePct float64
	eightBit     bool
	join         bool
	version      string
}

func defaultCliOptions() *cliOpts {
	return &cliOpts{
		crs:          "",
		maxDepth:     10,
		minPoints:    2000,
		maxPoints:    160000,
		resolution:   20,
		subsamplePct: 1,
		zOffset:      0,
		eightBit:     false,
		join:         false,
		version:      "1.0",
	}
}

func (c *cliOpts) validate() {
	if c.output == "" {
		log.Fatal("output flag must be set")
	}
	if c.maxDepth <= 1 || c.maxDepth > 20 {
		log.Fatal("depth should be between 1 and 20")
	}
	if c.minPoints < 1 {
		log.Fatal("min-points-per-tile should be at least 1")
	}
	if c.maxPoints < 0 {
		log.Fatal("max-points-per-tile should be at least 0 (0 means no limit)")
	}
	if c.resolution < 0.5 || c.resolution > 1000 {
		log.Fatal("resolution should be between 1 and 1000 meters")
	}
	if c.subsamplePct < 0.01 || c.subsamplePct > 1 {
		log.Fatal("subsample should be a value between 0.01 and 1")
	}
	if _, ok := version.Parse(c.version); !ok {
		log.Fatal("invalid tileset version, the only allowed values are '1.0' and '1.1'")
	}
}

func (c *cliOpts) print() {
	crsMsg := c.crs
	if c.crs == "" {
		crsMsg = "(autodetect from LAS metadata)"
	}
	fmt.Printf(`*** Execution settings:
- Source CRS: %s,
- Max Depth: %d,
- Resolution: %f meters,
- Min Points per tile: %d,
- Max Points per tile: %d
- Z-Offset: %f meters,
- 8Bit Color: %v
- Join Clouds: %v
- Tileset Version: %v

`, crsMsg, c.maxDepth, c.resolution, c.minPoints, c.maxPoints, c.zOffset, c.eightBit, c.join, c.version)
}

func (c *cliOpts) getTilerOptions() *tiler.TilerOptions {
	c.validate()
	v, ok := version.Parse(c.version)
	if !ok {
		log.Fatal("unrecongnized tileset version")
	}
	mutators := []mutator.Mutator{
		mutator.NewZOffset(float32(c.zOffset)),
	}
	if c.subsamplePct < 1 {
		mutators = append(mutators, mutator.NewSubsampler(c.subsamplePct))
	}
	return tiler.NewTilerOptions(
		tiler.WithEightBitColors(c.eightBit),
		tiler.WithMutators(mutators),
		tiler.WithGridSize(c.resolution),
		tiler.WithMaxDepth(c.maxDepth),
		tiler.WithMinPointsPerTile(c.minPoints),
		tiler.WithMaxPointsPerTile(c.maxPoints),
		tiler.WithCallback(eventListener),
		tiler.WithTilesetVersion(v),
	)
}

func fileCommand(opts *cliOpts, filepath string) {
	t, err := tilerProvider()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("*** Mode: File, process LAS file at %s\n", filepath)
	opts.print()
	tilerOpts := opts.getTilerOptions()
	crs := opts.crs
	if code, err := strconv.Atoi(crs); err == nil {
		crs = fmt.Sprintf("EPSG:%d", code)
	}
	runnable := func(ctx context.Context) error {
		return t.ProcessFiles([]string{filepath}, opts.output, crs, tilerOpts, ctx)
	}
	launch(runnable)
}

func folderCommand(opts *cliOpts, folderpath string) {
	t, err := tilerProvider()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("*** Mode: Folder, process all files in %s\n", folderpath)
	opts.print()
	tilerOpts := opts.getTilerOptions()
	crs := opts.crs
	if code, err := strconv.Atoi(crs); err == nil {
		crs = fmt.Sprintf("EPSG:%d", code)
	}
	runnable := func(ctx context.Context) error {
		if opts.join {
			files, err := utils.FindLasFilesInFolder(folderpath)
			if err != nil {
				return err
			}
			return t.ProcessFiles(files, opts.output, crs, tilerOpts, ctx)
		}
		return t.ProcessFolder(folderpath, opts.output, crs, tilerOpts, ctx)
	}
	launch(runnable)
}

func launch(function func(ctx context.Context) error) {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := function(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()
	wg.Wait()
}

func eventListener(e tiler.TilerEvent, filename string, elapsed int64, msg string) {
	fmt.Printf("[%s] [%s] %s\n", time.Now().UTC().Format("2006-01-02 15:04:05.000"), filename, msg)
}

func printBanner() {
	banner := strings.ReplaceAll(logo, "YYYY", strconv.Itoa(time.Now().Year()))
	banner = strings.ReplaceAll(banner, "ZZZZ", getVersion())
	fmt.Println(banner)
}
