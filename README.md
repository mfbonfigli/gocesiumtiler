# Go Cesium Point Cloud Tiler
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
##### Version 1.0.2 
* Fixed bug preventing tileset.json from being generated if only one pnts is created

##### Version 1.0.1 
* Fixed a crash occurring when converting point clouds without executing any coordinate system conversion.

##### Version 1.0.0 release
* First public release

## Precompiled Binaries
Along with the source code a prebuilt binary for Windows x64 is provided for each release of the tool in the github page.
Binaries for other systems at the moment are not provided.

## Installing
To get started with development just clone the repository. 

When launching a build with `go build` go modules will retrieve the required dependencies. 

As the project and its dependencies make use of C code, under windows you should also have GCC compiler installed and available
in the PATH environment variable. More information on cgo compiler are available [here](https://github.com/golang/go/wiki/cgo).

Under linux you will have to have `gcc` installed. Also make sure go is configured to pass the correct flags to gcc. In particular if you encounter compilation errors similar to `undefined reference to 'sqrt'` it means that it is not linking the standard math libraries. A way to fix this is to add `-lm` to the `CGO_LDFLAGS`environment variable, for example by running `export CGO_LDFLAGS="-g -O2 -lm"`.
## Usage

<b>The code expects to find a copy of the [static](static) folder in the same path where the compiled executable runs.</b>

To run just execute the binary tool with the appropriate flags.

It is suggested to try use the `-hq` flag as in most scenarios it does not slow down too much the tiling
process but it produces tiles that have better quality. One should experiment to decide whether it is worth using or not.

To show help run:
```
gocesiumtiler -help
```

### Flags

```
-input=<path>           input las file or folder containing las files. Required.
-output=<path>          output folder where to write cesium 3d tiles output. Required.
-srid=<epsg-code-no>    epsg code number of input coordinates (e.g. 4326 for EPSG:4326) [default: 4326]
-zoffset=<m>            vertical offset to apply to points, in meters [default: 0]
-maxpts=<n>             maximum number of points per each tile [default: 50000]
-geoid                  enables the geoid to ellipsoid elevation conversion
-folder                 enables the processing of all files in input folder
-recursive              if folder processing is enabled, recursively processes all LAS files found in subfolders
-silent                 suppresses all non error messages
-timestamp              adds a timestamp to console messages
-hq                     enables the use of a higher quality (but slightly slower) point sampling algorithm.
-help                   prints the help
```

### Usage examples:

Recursively convert all LAS files in folder `C:\las`, write output tilesets in folder `C:\out`, assume LAS input coordinates expressed 
in EPSG:32633, convert elevation from above the geoid to above the ellipsoid and use higher quality sampling algorithm:

```
gocesiumtiler -input=C:\las -output=C:\out -srid=32633 -geoid -folder -recursive -hq
```

Recursively convert all LAS files in `C:\las\file.las`, write output tileset in folder `C:\out`, assume input coordinates
expressed in EPSG:4326, apply an offset of 10 meters to elevation of points and allow to store up to 100000 points per tile:

```
gocesiumtiler -input=C:\las\file.las -output=C:\out -zoffset=10 -maxpts=100000
```

## Future work and support

Further work needs to be done, such as: 

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
