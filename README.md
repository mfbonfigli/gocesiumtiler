# Go Cesium Point Cloud Tiler
![Build Status](https://codebuild.eu-west-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoibSt1bmxTSGZTNHlLaS81d3ZEM3EyQXZIUXh4U0g4VjdtZUIwaUNMeEswaU5nc3A4YzZQQ2JiQ1pJbkFaVWxrcmF0TEFnRkM5WFFIbTVKQmtYSkVBTy9vPSIsIml2UGFyYW1ldGVyU3BlYyI6IjZjZ1NodXJzczJ0MlJja2UiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=master)
```
                                             _   _ _
  __ _  ___   ___ ___  ___(_)_   _ _ __ ___ | |_(_) | ___ _ __
 / _  |/ _ \ / __/ _ \/ __| | | | | '_   _ \| __| | |/ _ \ '__|
| (_| | (_) | (_|  __/\__ \ | |_| | | | | | | |_| | |  __/ |
 \__, |\___/ \___\___||___/_|\__,_|_| |_| |_|\__|_|_|\___|_|
  __| | A Cesium Point Cloud tile generator written in golang
 |___/ 
```



Go Cesium Point Cloud Tiler is a tool to convert point cloud stored as LAS files to Cesium.js 3D tiles ready to be
streamed, automatically generating the appropriate level of details and including additional information for each point 
such as color, laser intensity and classification.   

## Features
Go Cesium Point Cloud Tiler automatically handles coordinate conversion to the format required by Cesium and can also 
convert the elevation measured above the geoid to the elevation above the ellipsoid as by Cesium requirements. 
The tool uses the version 4.9.2 of the well-known Proj.4 library to handle coordinate conversion. The input SRID is
specified by just providing the relative EPSG code, an internal dictionary converts it to the corresponding proj4 
projection string.

Speed is a major concern for this tool, thus it has been chosen to store the data completely in memory. If you don't 
have enough memory the tool will fail, so if you have really big LAS files and not enough RAM it is advised to split 
the LAS in smaller chunks to be processed separately.

Information on point intensity and classification is stored in the output tileset Batch Table under the 
propeties named `INTENSITY` and `CLASSIFICATION`.


## Changelog
##### Version 1.2.2
* Fixed a bug in parsing RGB colors for LAS files having point records of non standard length.
* Fixed a bug when processing multiple files (this was not really fixed in the previous version).

##### Version 1.2.1
* Added option to support LAS files with 8bit color depth.
* Fixed a bug when processing multiple files.
* Fixed a bug in computing the geometric error of the lowest level of detail tile.

##### Version 1.2.0 
* Added support for `REPLACE` refine mode. Default refine mode is still `ADD`.

##### Version 1.1.1 
* Fixed a bug that prevented the executable to be used on computers other than the one where the code was built.

##### Version 1.1.0 
* Added a new tiling algorithm that greatly improves the output quality.

##### Version 1.0.3 
* Added shorthand versions of input flags and a new intro logo. Also a major code refactoring has happened behind the scenes. 

##### Version 1.0.2 
* Fixed bug preventing tileset.json from being generated if only one pnts is created

##### Version 1.0.1 
* Fixed a crash occurring when converting point clouds without executing any coordinate system conversion.

##### Version 1.0.0 release
* First public release

## Precompiled Binaries
Along with the source code a prebuilt binary for Windows x64 is provided for each release of the tool in the github page.
Binaries for other systems at the moment are not provided.

## Environment setup and compiling from sources
To get started with development just clone the repository. 

When launching a build with `go build` go modules will retrieve the required dependencies. 

As the project and its dependencies make use of C code, under windows you should also have GCC compiler installed and available
in the PATH environment variable. More information on cgo compiler are available [here](https://github.com/golang/go/wiki/cgo).

Under linux you will have to have `gcc` installed. Also make sure go is configured to pass the correct flags to gcc. In particular if you encounter compilation errors similar to `undefined reference to 'sqrt'` it means that it is not linking the standard math libraries. A way to fix this is to add `-lm` to the `CGO_LDFLAGS`environment variable, for example by running `export CGO_LDFLAGS="-g -O2 -lm"`.

To launch the tests use the command `go test ./test/... -v`.

## Usage

<b>The code expects to find a copy of the [static](assets) folder in the same path where the compiled executable runs.</b>

> Alternatively, from version 1.1.1 you can also specify the assets folder location (i.e. the folder that contains the `assets` folder) 
by setting the `GOCESIUMTILER_WORKDIR` environment variable in your system.

To run just execute the binary tool with the appropriate flags.

There are various algorithms selectable. It is highly suggested to use the newer "grid" algorithm, which is the default one.
Other possible choices are "random" and "randombox", which however are deprecated even though they might turn out to be slightly
faster in common scenarios.

To show help run:
```
gocesiumtiler -help
```

### Flags

```
  -8bit                 Assumes the input LAS has colors encoded in eight bit format. Default is false (LAS has 16 bit color depth)
  -a string             Sets the algorithm to use. Must be one of Grid,Random,RandomBox. Grid algorithm is highly suggested, others are deprecated and will be removed in future versions. (shorthand for algorithm) (default "grid")
  -algorithm string     Sets the algorithm to use. Must be one of Grid,Random,RandomBox. Grid algorithm is highly suggested, others are deprecated and will be removed in future versions. (default "grid")
  -b                    Assumes the input LAS has colors encoded in eight bit format. Default is false (LAS has 16 bit color depth). (shorthand for -8bit)
  -e int                EPSG srid code of input points. (shorthand for srid) (default 4326)
  -f                    Enables processing of all las files from input folder. Input must be a folder if specified (shorthand for folder)
  -folder               Enables processing of all las files from input folder. Input must be a folder if specified
  -g                    Enables Geoid to Ellipsoid elevation correction. Use this flag if your input LAS files have Z coordinates specified relative to the Earth geoid rather than to the standard ellipsoid. (shorthand for geoid)
  -geoid                Enables Geoid to Ellipsoid elevation correction. Use this flag if your input LAS files have Z coordinates specified relative to the Earth geoid rather than to the standard ellipsoid.
  -grid-max-size float  Max cell size in meters for the grid algorithm. It roughly represents the max spacing between any two samples.  (default 5)
  -grid-min-size float  Min cell size in meters for the grid algorithm. It roughly represents the minimum possible size of a 3d tile.  (default 0.15)
  -h                    Displays this help. (shorthand for help)
  -help                 Displays this help.
  -i string             Specifies the input las file/folder. (shorthand for input)
  -input string         Specifies the input las file/folder.
  -m int                Max number of points per tile for the Random and RandomBox algorithms. (shorthand for maxpts) (default 50000)
  -maxpts int           Max number of points per tile for the Random and RandomBox algorithms. (default 50000)
  -n float              Min cell size in meters for the grid algorithm. It roughly represents the minimum possible size of a 3d tile.  (shorthand for grid-min-size) (default 0.15)
  -o string             Specifies the output folder where to write the tileset data. (shorthand for output)
  -output string        Specifies the output folder where to write the tileset data.
  -r                    Enables recursive lookup for all .las files inside the subfolders (shorthand for recursive)
  -recursive            Enables recursive lookup for all .las files inside the subfolders
  -refine-mode          Type of refine mode, can be 'ADD' or 'REPLACE'. 'ADD' means that child tiles will not contain the parent tiles points. 'REPLACE' means that they will also contain the parent tiles points. ADD implies less disk space but more network overhead when fetching the data, REPLACE is the opposite. (default "ADD")
  -s                    Use to suppress all the non-error messages. (shorthand for silent)
  -silent               Use to suppress all the non-error messages.
  -srid int             EPSG srid code of input points. (default 4326)
  -t                    Adds timestamp to log messages. (shorthand for timestamp)
  -timestamp            Adds timestamp to log messages.
  -v                    Displays the version of gocesiumtiler. (shorthand for version)
  -version              Displays the version of gocesiumtiler.
  -x float              Max cell size in meters for the grid algorithm. It roughly represents the max spacing between any two samples.  (shorthand for grid-max-size) (default 5)
  -z float              Vertical offset to apply to points, in meters. (shorthand for zoffset)
  -zoffset float        Vertical offset to apply to points, in meters.
```

Note: the "hq" flag present in versions <= 1.0.3 has been removed and replaced by the "randombox" setting for the `-algorithm` flag.


### Usage examples:

Recursively convert all LAS files in folder `C:\las`, write output tilesets in folder `C:\out`, assume LAS input coordinates expressed 
in EPSG:32633, convert elevation from above the geoid to above the ellipsoid and use the default grid sampling algorithm:

```
gocesiumtiler -input=C:\las -output=C:\out -srid=32633 -geoid -folder -recursive 
```
or, using the shorthand notation:
```
gocesiumtiler -i C:\las -o C:\out -e 32633 -g -f -r
```

Recursively convert all LAS files in `C:\las\file.las`, write output tileset in folder `C:\out`, assume input coordinates
expressed in EPSG:4326, apply an offset of 10 meters to elevation of points and allow to store up to 100000 points per tile
using the "randombox" algorithm:

```
gocesiumtiler -input=C:\las\file.las -output=C:\out -zoffset=10 -maxpts=100000 -algorithm=randombox
```
or, using the shorthand notation:

```
gocesiumtiler -i C:\las\file.las -o C:\out -z 10 -m 100000 -a randombox
```

### Algorithms
As of now all the algorithms provided in the tool divide the space in an octree (i.e. a partition  of 8 octants recursively subdivided in octants as well).
Every octant contains points plus 8 children, which are octants as well. These children octants might contain points and octants as well,
in a recursive fashion.

The difference between the algorithms is tied to how they choose the points to store in each level of the octree.

- **Grid algorithm**
This algorithm samples the space in a grid structure. For a `grid-max-size=5m` it stores one point every 5x5x5m box in the space,
the closest one to the cell center. All the other points (the ones not closest to the 5x5x5 3D grid intersections) are sent to the
children levels, where the grid halves in size, and the process is repeated until either the points are all stored or the grid size has 
reached the `grid-min-size` setting. `grid-max-size` setting should be set accordingly to the input cloud size. A value that is too small
might result in very dense tiles at higher LODs, a value that is too big might result in very few points stored ad higher LODs and 
a highly nested tree structure.

- **Random algorithm** 
This algorithm simply shuffles all the points in the point cloud and picks at random up to `maxpts` points for each octree node.
Shuffling allows to uniformely represent the overall shape of the point cloud, however this might imply that some details
are not adequately represented at higher distances (lower Level of Details). 

- **RandomBox algorithm** 
This algorithm is very similar to the Random algorithm but with one difference. Points are divided into bins of a definite, small, size
and the bins are randomly shuffled. Then the points are picked one by one from each bin. This ensures that points are randomly 
distributed but also that all areas of space, even the ones with fewer points, are equally likely to be represented at higher level of details. 

### Refine Modes
Cesium tilesets can have two different *refine* settings, `ADD` and `REPLACE`, briefly explained as follow:
- `ADD` refine mode means that a certain tile will contain only the points not already contained in the parent tiles. This 
means that to render the points in the tile all parent tiles points must be fetched and loaded. In other words this mode 
tells Cesium to display the Tile plus its parent.
- `REPLACE` refine mode means that a certain tile will be self-contained, i.e. its `.pnts` file will contain ALL the points
needed for rendering, including those already belonging to the parent tiles. In other words this mode instructs Cesium
to only visualize this tile and not the points contained in its parent too.

As a consequence in gocesiumtiler `ADD` mode results in smalled disk space as there are no duplicate points stored across
LODs. Plus it is faster. In theory `REPLACE` mode might however be more memory and network efficient as Cesium can only visualize and load the
required tile for the given LOD and not also the parent tiles, but this highly depends on how the Cesium Viewer settings
have been configured. For this reason `ADD` mode is the default and suggested one, but one can specify `REPLACE` mode 
by using `-refine-mode REPLACE`.

## Precompiled Binaries
Along with the source code a prebuilt binary for Windows x64 is provided for each release of the tool in the github page.
Binaries for other systems at the moment are not provided.

## Future work and support

Further work needs to be done, such as: 
- Improving the test coverage and creating a test pipeline.
- Adding an automatic setting to estabilish `grid-max-size` and `grid-min-size` values for the grid algorithm based on 
the input point cloud parameters.
- Adding support for non-metric units for elevations.
- Integration with the [Draco](https://github.com/google/draco) compression library
- Upgrading of the Proj4 library to versions newer than 4.9.2
- Optimizations to reduce the memory footprint so to process bigger LAS files
- Develop new sampling algorithms to increase the quality of the point cloud and/or processing speed
 
Contributors and their ideas are welcome.

If you have questions you can contact me at <m.federico.bonfigli@gmail.com>

## Versioning

This library uses [SemVer](http://semver.org/) for versioning. 
For the versions available, see the [tags on this repository](https://github.com/mfbonfigli/gocesiumtiler/tags). 

## Credits

**Massimo Federico Bonfigli** -  [Github](https://github.com/mfbonfigli)

## License

This project is licensed under the GNU Lesser GPL v.3 License - see the [LICENSE.md](LICENSE.md) file for details.

The software uses third party code and libraries. Their licenses can be found in
[LICENSE-3RD-PARTIES.md](LICENSE-3RD-PARTIES.md) file.

## Acknowledgments

* Cesium JS library [github](https://github.com/AnalyticalGraphicsInc/cesium)
* TUM-GIS cesium point cloud generator [github](https://github.com/tum-gis/cesium-point-cloud-generator)
* Simon Hege's golang bindings for Proj.4 library [github](https://github.com/xeonx/proj4)
* John Lindsay go library for reading LAS files [lidario](https://github.com/xeonx/proj4)
* Sean Barbeau Java porting of Geotools EarthGravitationalModel code [github](https://github.com/barbeau/earth-gravitational-model)
