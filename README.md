# gocesiumtiler
![Build Status](https://codebuild.eu-west-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoiTW5nSHpJcGJTQjJEekVQL2JsbUJWeVhmUHNZMFhsRGh6TXdPaGxSSlhHVkFCQ1RUUVM0YjJ6dWkyNmJaQmVKdE94RDVzNEhwbHZaOHJGakpCMkJCUG00PSIsIml2UGFyYW1ldGVyU3BlYyI6Im9COW9CVm9FM3ljQlR4RmQiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=v2)
```
                                             _   _ _
  __ _  ___   ___ ___  ___(_)_   _ _ __ ___ | |_(_) | ___ _ __
 / _  |/ _ \ / __/ _ \/ __| | | | | '_   _ \| __| | |/ _ \ '__|
| (_| | (_) | (_|  __/\__ \ | |_| | | | | | | |_| | |  __/ |
 \__, |\___/ \___\___||___/_|\__,_|_| |_| |_|\__|_|_|\___|_|
  __| | A Cesium Point Cloud tile generator written in golang
 |___/ 
```


**gocesiumtiler** is a tool to convert point cloud stored as LAS files to Cesium.js 3D tiles ready to be
streamed, automatically generating the appropriate level of details and including additional information for each point 
such as color, laser intensity and classification.   

## What's new: V2.0.0
gocesiumtiler V2.0.0 has been released introducing several important improvements over V1. Please refer to the changelog for a full list of changes.

This release is backward incompatible compared to V1 and also deprecates several options, most of which were not of much interest.
Some of these might be added in future minor updates of V2.
- Refine mode "REPLACE" is currently unavailable in V2
- Recursive options is currently unavailable in V2
- Algorithm cannot be chosen anymore: grid sampling is the only one available and is implicitly used

## Features
gocesiumtiler V2 offers the following features:

- Supports LAS 1.4 and writes Intensity and Classification attributes into the final point cloud
- Performs automatic coordinate conversion without any external library dependency
- Allows setting a custom elevation offset for the point clouds points
- Can automatically subsample the input point clouds
- Can merge multiple LAS files into a single tileset automatically
- Supports both 3D Tiles Specs 1.0 (.pnts) and (experimentally) 3D Tiles v1.1 (glTF/GLB assets)
- Fast: Uses all available cores and minimizes disk operations
- High quality sampling via an uniform sampling scheme
- Can read LAS with colors encoded in 8bit color space rather than 16bit
- Can be used in other golang programs as a library with a simple to use interface
- Supports the programmatic definition of custom mutators to manipulate points and attributes
- Can automatically extract CRS metadata from LAS (if present) or a CRS can be provided in form of EPSG code, Proj4 string or WKT definition.
- Conversion is done via well knwon Proj 9.5.0 library (for some projections the relevant Proj grid files should be installed separately)
- Uses a "ADD" refine method, which minimizes redundant data across tiles


**New from v2.0.0-gamma**: Convertion between EGM and WGS84 elevations is now delegated to Proj. Just make sure to specify the correct input vertical datum and make sure the `share` folder contains the required vertical grids. These are not included but can be downloaded from the [Proj CDN](https://cdn.proj.org/)

The currently tool uses the version 9.5.0 of the well-known Proj.4 library to handle coordinate conversion. The input CRS specified by just providing the relative EPSG code, a Proj4 string or WKT text, or letting the library figure it out automatically from the LAS metadata.

Speed is a major concern for this tool, thus it has been chosen to store the data completely in memory. If you don't 
have enough memory the tool will fail, so if you have really big LAS files and not enough RAM it is advised to split 
the LAS in smaller chunks to be processed separately.

Information on point intensity and classification is stored in the output tileset Batch Table under the 
propeties named `INTENSITY` and `CLASSIFICATION`.

## Demo
You can preview a couple of tilesets generated from this tool at this [website](https://d39maarsub1d2t.cloudfront.net/index.html)


## Changelog
##### Version 2.0.1
* Resolved a bug that prevented the correct functioning of the join flag when processing multiple LAS files from a folder.

##### Version 2.0.0
* Most of the code has been rewritten from the ground up, achieving much faster tiling with lower memory usage
* Uses Proj v9.5.0: all projections supported by the Proj library are automatically supported by gocesiumtiler.
* Able to autodetect CRS from GeoTIFF or WKT metadata embedded in the LAS VLRs.
* Experimentally supports 3D Tiles v1.1 (GLTF) in addition to v1.0 (PNTS)
* Allows merging multiple LAS files into a single 3D tile
* New options to fine tune the sampling quality: min points per tile, resolution and max tree depth
* Ready to be used as part of other go packages with an easy to use interface
* Vends the mutator interface, to programmatically alter points and point attributes as they are read
* Can automatically subsample input point clouds using a simple random sampling scheme
* Points are now expressed relative to a local cartesian reference system with Z-up. This results in tighter tile bounds, and
  now shaders can utilize the local Z coordinate for elevation-based shading, which was not possible before.
* Much higher unit test coverage
* The tiler library can be used into external golang projects
* The CLI has been rewritten from scratch, many options changed name or were added/removed
* `GOCESIUMTILER_WORKDIR` has been deprecated

**Note**: The final V2.0.0 version significantly differs from "pre-release" versions V2.0.0 alpha, beta and gamma. The V2.0.0 final changelog consolidates all changes from V1.

## Precompiled Binaries
Along with the source code a prebuilt binary for Windows x64 and Linux x64 is provided for each release of the tool in the github page.
Binaries for other systems at the moment are not provided.

## Environment setup and compiling from sources
Please note that the development setup has changed considerably from V2.0.0 gamma and now leverages Docker for reproducible builds. 

In general local development requires:

- Downloading the Proj sources
- Building Proj as a static library
- Building gocesiumtiler linking it statically to Proj and its dependencies

Please refer to the [DEVELOPMENT.md](DEVELOPMENT.md) file for further info on how to setup a local development environment in both Windows and Linux.

## Installation instructions

1. Download the latest version of the executable from the Releases section in github and unzip it
1. Make sure the executable is in the same folder where the `share` folder is. 
2. (Optional) If you need special grids to convert your data (e.g. in case of EGM to WGS84 elevation conversion or some less common projections), please download the Proj Data grids from the [Proj CDN](https://cdn.proj.org/) and unpack them in the `share` folder. 
3. Execute the binary tool with the appropriate flags.

## CLI Usage

To run just execute the binary tool with the appropriate flags.

To show the help run:
```
gocesiumtiler -help
```

### Commands
There are two commands, `file` and `folder`:

* `gocesiumtiler file { flags } myfile.las`: Converts `myfile.las` into a Cesium 3D point cloud using the flags passed in input (see below).
* `gocesiumtiler folder { flags } myfolder`: Finds all LAS files into `myfolder` and convers them into one or more Cesium 3D Point clouds using the flags passed as input (see below).S

### Flags

#### Common flags
These flags are applicable to both the `file` and the `folder` commands
```
   --out value, -o value                  full path of the output folder where to save the resulting Cesium tilesets
   --crs value, --epsg value, -e value    String representing the input CRS. For example, and EPSG code like EPSG:4326 or EPSG:28355+5773 or a generic Proj4 or WKT string. Bare numbers will be interpreted as EPSG codes. If empty the system will attempt to autodetect the CRS from the LAS metadata. In case of multiple LAS files, the CRS must be consistent else an error will be thrown.
   --resolution value, -r value           minimum resolution of the 3d tiles, in meters. approximately represets the maximum sampling distance between any two points at the lowest level of detail (default: 20)
   --z-offset value, -z value             z offset to apply to the point, in meters. only use it if the input elevation is referred to the WGS84 ellipsoid or geoid (default: 0)
   --depth value, -d value                maximum depth of the output tree. (default: 10)
   --version, -v value                    output tileset version (default: 1.0, can be 1.1 for glb content tiles)
   --min-points-per-tile value, -m value  minimum number of points to enforce in each 3D tile (default: 5000)
   --8-bit                                set to interpret the input points color as part of a 8bit color space (default: false)  
   --subsample value                      Approximate percent of points to keep in the final point cloud, between 0.01 (1%) and 1 (100%) (default: 1)
   --help, -h                             show help
```

#### Folder command flags
These commands are specific to the `folder` command:
```
   --join, -j                             merge the input LAS files in the folder into a single cloud. The LAS files must have the same properties (CRS etc) (default: false)
```

#### A note on vertical coordinate conversion
Previous releases of gocesiumtiler had a dedicated flag for EGM to WGS84 ellipsoid elevation conversion. This has been deprecated and now the vertical datum conversion is fully delegated to Proj.
This means that to convert the vertical coordinates in case they are not referred to the WGS84 ellipsoid the input CRS definition needs to include the definition for the vertical datum.

If the LAS file includes a 3D CRS definition or separate Horizontal and Vertical CRS definitions gocesiumtiler will attempt to use those info automatically. Else if a vertical definition is not included, 
or the tool fails to detect the correct CRS, the right definition can be provided manually via the `--crs` flag.

This can be achieved for example by providing a composite CRS EPSG code. 

For example, if a LAS file contains coordinates in the `EPSG:32633` CRS with elevations referred to the EGM2008 (`EPSG:3855`), run the command with:

```
--crs EPSG:32633+3855
```

**NOTE:** This might require to install extra grid files in the `share` folder. In case of issues please make sure please to download the Proj Data grids from the [Proj CDN](https://cdn.proj.org/) and save them in the `share` folder.

### Usage examples:

#### Example 1

Convert all LAS files in folder `C:\las`, write output tilesets in folder `C:\out`, assume LAS input coordinates expressed 
in EPSG:32633, with elevations referred to the EGM2008 (EPSG 3855). Use a resolution of 10 meters, create tiles with minimum 1000 points
and enforce a maximum tree depth of 12:

```
gocesiumtiler folder -out C:\out -crs EPSG:32633+3855 -resolution 10 -min-points-per-tile 1000 -depth 12 C:\las
```

or, using the shorthand notation:

```
gocesiumtiler folder -o C:\out -e EPSG:32633+3855 -r 10 -m 1000 -d 12 C:\las
```

#### Example 2

Like Example 1 but merge all LAS files in `C:\las` into a single 3D Tileset

```
gocesiumtiler folder -out C:\out -crs EPSG:32633+3855 -resolution 10 -min-points-per-tile 1000 -depth 12 -join C:\las
```

or, using the shorthand notation:
```
gocesiumtiler folder -o C:\out -e EPSG:32633+3855 -r 10 -m 1000 -d 12 -j C:\las
```

#### Example 3

Convert a single LAS file at `C:\las\file.las`, write the output tileset in the folder `C:\out`, use the system defaults and let gocesiumtiler extract the CRS metadata from the LAS.

```
gocesiumtiler file -out C:\out C:\las\file.las
```

or, using the shorthand notation:

```
gocesiumtiler file -o C:\out -e 32633 C:\las\file.las
```

## Library Usage in other GO programs

To use the tiler in other go programs just:

```
go get github.com/mfbonfigli/gocesiumtiler/v2
```

Then instantiate a tiler object and launch it either via `ProcessFiles` or `ProcessFolder` passing in the desired processing options.

A mnimal example is:

```
package main

import (
	"context"
	"log"

	"github.com/mfbonfigli/gocesiumtiler/v2/tiler"
)

func main() {
	t, err := tiler.NewGoCesiumTiler()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.TODO()
	err = t.ProcessFiles([]string{"myinput.las"}, "/tmp/myoutput", "EPSG:32632", tiler.NewTilerOptions(
		tiler.WithEightBitColors(true),
		tiler.WithWorkerNumber(2),
		tiler.WithMaxDepth(5),
	), ctx)
	if err != nil {
		log.Fatal(err)
	}
}
```

Note that you will require to use `cgo` for the compilation, for how to setup the build environment please refer to the [DEVELOPMENT.md](DEVELOPMENT.md). 

### Mutators

gocesiumtiler from version 2.0.0 final offers the concept of **mutators**. Mutators are implementations of the `mutator.Mutator` interface 
and can be used to manipulate or discard input points. 

The library vends a `ZOffset` mutator to perform vertical traslation of point clouds and a `Subsampler` mutator to thin down the points in the output. 

Other possible uses of mutators (not yet built in into the library) could be, for example:
- Perform color corrections of points
- Colorize the points based on their Classification
- Cut point clouds discarding points outside of given bounds
- etc

To use the mutators just pass them to the tiler options:

```
mut := []mutator.Mutator{
   mutator.NewZOffset(10),
   mutator.NewSubsampler(0.5),
}

err = t.ProcessFiles([]string{"myinput.las"}, "/tmp/myoutput", "EPSG:32632", tiler.NewTilerOptions(
   tiler.WithEightBitColors(true),
   tiler.WithWorkerNumber(2),
   tiler.WithMaxDepth(5),
   tiler.WithMutators(mut),
), ctx)
```

To implement a custom mutator, implement the mutator interface:

```
Mutate(p model.Point, t model.Transform) (model.Point, bool)
```

The Mutate function receives as input the original point and a Transform object that can be used to trasform from the local CRS to the global EPSG 4978 CRS and back. 
The output of the Mutate function is the, eventually manipulated, point and a boolean which should be true if the point should appear in the final point cloud, false otherwise.

**Note: mutators must be goroutine safe.**

#### Example of a custom mutator: coloring points by class

```
type ClassBasedColorMutator struct {}

func (m *ClassBasedColorMutator) Mutate(pt model.Point, localToGlobal model.Transform) (model.Point, bool) {
	switch pt.Classification {
	case 2:
		// ground > brown
		pt.R, pt.G, pt.B = 128, 0, 0
	case 3:
		// low vegetation > dark green
		pt.R, pt.G, pt.B = 0, 60, 0
	case 4:
		// medium vegetation > green
		pt.R, pt.G, pt.B = 0, 130, 0
	case 5:
		// high vegetation > light green
		pt.R, pt.G, pt.B = 30, 170, 30
	case 6:
		// building > orange
		pt.R, pt.G, pt.B = 200, 100, 0
	case 9:
		// water > blue
		pt.R, pt.G, pt.B = 5, 170, 200
	default:
		// all other classes > grey
		pt.R, pt.G, pt.B = 30, 30, 30
	}
	return pt, true
}
```

## Algorithms

The sampling occurs using a hybrid, lazy octree data structure. The algorithm works as follows:
1. All points are stored in a linked tree backed by an underlying array: this provides efficient list manipulations operations (splitting, adding, removing) and avoids
 dynamic allocations of slices. The coordinates are internally converted to EPSG 4978 and then to a local CRS with Z-axis normal to the WGS84 ellipsoid. The backing array helps with CPU cache friendliness and helps achieving measurable speed improvements of up to 20% compared to a standard linked list.
2. An octree cell is created. Every cell stores N points (variable). A cell also has a grid spacing property. The root node has a spacing set to the
 provided resolution. 
3. The points are traversed. Each point will fall into one of the cells the space has been divided by the given grid spacing. If the point is 
 the closest one the the center of the cell, it's taken as new closest for that cell, else it's discarded and parked into a list of the Node octant it belongs to.
4. When all points are traversed the points closes to the cells the space has been partitioned in will be the points for the current tree node. All others are parked.
If an octant however has a number of parked points that is less than the min-points-per-node, the parked points for that octant are rolled up to the current node. Similarly 
if the current node has a depth equal to the configured max depth.
5. Whenever the children are retrieved, the previously parked points are used to create child nodes on demand using the same algorithm, lazily.
6. The points are written in the final artifacts with coordinates relative to the local CRS, however a global transform is applied at the tileset root to convert points back to the EPSG 4978 CRS required by Cesium.

## Precompiled Binaries
Along with the source code, a prebuilt binary for both Linux and Windows x64 is provided for each release of the tool in the github page.

## Future work and support

Further work needs to be done, such as: 
- Statically build and link Proj9.5.0 with cURL support enabled
- Add support for point cloud compression
- Add support for LAZ (compressed LAS) files
- Make Intensity and Classification optional attributes of the output cloud to save disk space

Contributors and their ideas are welcome.

If you have questions you can contact me at <m.federico.bonfigli@gmail.com>

## Versioning

This library uses [SemVer](http://semver.org/) for versioning. 
For the versions available, see the [tags on this repository](https://github.com/mfbonfigli/gocesiumtiler/v2/tags). 

## Credits

**Massimo Federico Bonfigli** -  [Github](https://github.com/mfbonfigli)

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE.md](LICENSE.md) file for details.

The software uses third party code and libraries. Their licenses can be found in
[LICENSE-3RD-PARTIES.md](LICENSE-3RD-PARTIES.md) file.

## Acknowledgments

* Cesium JS library [github](https://github.com/AnalyticalGraphicsInc/cesium)
* Tom Payne's golang bindings for the Proj library [github](https://github.com/twpayne/go-proj)
* John Lindsay go library for reading LAS files [lidario](https://github.com/jblindsay/lidario)
* TUM-GIS cesium point cloud generator [github](https://github.com/tum-gis/cesium-point-cloud-generator)
* Simon Hege's golang bindings for Proj.4 library [github](https://github.com/xeonx/proj4)
* Sean Barbeau Java porting of Geotools EarthGravitationalModel code [github](https://github.com/barbeau/earth-gravitational-model)

### License
Lhe library is released under the Mozilla Public License Version 2.0. See the [LICENSE.md](LICENSE.md) file for further info.
