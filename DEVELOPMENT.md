# gotiler Development

This repository contains the public `gotiler` CLI. The tiling engine lives in the public Go module [`github.com/mfbonfigli/gotiler-core`](https://github.com/mfbonfigli/gotiler-core).

## Reproducible Docker Builds

Docker is the recommended way to build release artifacts because it builds PROJ and the C/C++ dependencies in a known environment.

```bash
bash scripts/build.sh linux-amd64
bash scripts/build.sh linux-arm64
bash scripts/build.sh windows-amd64
bash scripts/build.sh all
```

Windows PowerShell:

```powershell
.\scripts\build.ps1 -Target windows-amd64
```

No private GitHub token is required.

## Local Windows Development

Local builds require Go, CGO, MinGW, pkg-config, vcpkg dependencies, and a static PROJ build.

Install MSYS2 packages:

```bash
pacman -S --noconfirm mingw-w64-x86_64-pkgconf
pacman -S --noconfirm mingw-w64-x86_64-gcc
pacman -S --noconfirm mingw-w64-x86_64-cmake
pacman -S --noconfirm mingw-w64-x86_64-sqlite3
```

Install vcpkg dependencies:

```powershell
git clone https://github.com/Microsoft/vcpkg.git C:\vcpkg
cd C:\vcpkg
.\bootstrap-vcpkg.bat -disableMetrics
.\vcpkg.exe install sqlite3[core,tool] tiff zlib --triplet=x64-mingw-static
```

Build PROJ statically, following the Dockerfile for the canonical version and flags. The shape is:

```bash
cmake -DCMAKE_TOOLCHAIN_FILE=C:/vcpkg/scripts/buildsystems/vcpkg.cmake \
  -DVCPKG_TARGET_TRIPLET=x64-mingw-static \
  -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc \
  -DCMAKE_CXX_COMPILER=x86_64-w64-mingw32-g++ \
  -DCMAKE_INSTALL_PREFIX=/usr/local/ \
  -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_APPS=OFF \
  -DBUILD_SHARED_LIBS=OFF \
  -DENABLE_CURL=OFF \
  -DENABLE_TIFF=ON \
  -DEMBED_PROJ_DATA_PATH=OFF \
  -DBUILD_TESTING=OFF ..
cmake --build . --config Release -j 8
cmake --build . --target install -j 8
```

Then build:

```powershell
$env:PKG_CONFIG_PATH="C:\usr\local\lib\pkgconfig;C:\vcpkg\installed\x64-mingw-static\lib\pkgconfig"
$env:CC="x86_64-w64-mingw32-gcc"
$env:CGO_ENABLED="1"
$env:CGO_LDFLAGS="-L/c/vcpkg/installed/x64-mingw-static/lib -g -O2 -static -lstdc++ -lsqlite3 -ltiff -lz -ljpeg -llzma -lm"

go build -o ./bin/gotiler.exe ./cmd/main.go
go test ./...
```

## Local Linux Development

For Ubuntu-like environments, follow the Dockerfile steps:

1. Install build tools, CMake, pkg-config, SQLite, TIFF, and GCC.
2. Bootstrap vcpkg.
3. Install `sqlite3[core,tool]` and `tiff`.
4. Build PROJ statically with the same flags used by the Dockerfile.
5. Export `PKG_CONFIG_PATH`, `CGO_ENABLED`, `CGO_LDFLAGS`, and `PROJ_DATA`.
6. Run `go test ./...`.

## Updating gotiler-core

Because `gotiler-core` is public, updating is a normal Go module operation:

```bash
go get github.com/mfbonfigli/gotiler-core@main
go mod tidy
```

For local development against a sibling checkout:

```bash
go mod edit -replace github.com/mfbonfigli/gotiler-core=C:\Users\bonfi\workplace\gotiler-core
```

Remove it before committing unless the replacement is intentional:

```bash
go mod edit -dropreplace github.com/mfbonfigli/gotiler-core
go mod tidy
```
