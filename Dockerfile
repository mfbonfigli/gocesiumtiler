##################################
#        BASE BUILD IMAGE        #
##################################
FROM ubuntu:latest AS builder-base

# Change these two arguments to change the version of go and PROJ
ARG GO_VERSION="1.23.2"
ARG PROJ_VERSION="proj-9.5.0"

# build variable, no impact on the final artifacts
ARG PROJECT_FOLDER="/usr/src/gocesiumtiler"

# install essential tools
# partly taken from https://github.com/OSGeo/PROJ/blob/master/Dockerfile
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --fix-missing --no-install-recommends \
    apt-transport-https software-properties-common ca-certificates wget zip unzip curl tar pkg-config \
    git cmake make sqlite3 libsqlite3-dev \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# install powershell, required for vcpkg 
# taken from https://learn.microsoft.com/en-us/powershell/scripting/install/install-ubuntu?view=powershell-7.4
RUN . /etc/os-release && wget -q "https://packages.microsoft.com/config/ubuntu/$VERSION_ID/packages-microsoft-prod.deb"
RUN dpkg -i packages-microsoft-prod.deb
RUN rm packages-microsoft-prod.deb
RUN apt-get update
RUN apt-get install -y powershell
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# install vcpkg to manage packages and dependencies
WORKDIR /vcpkg
RUN git clone https://github.com/Microsoft/vcpkg.git "/vcpkg"
RUN ./bootstrap-vcpkg.sh -disableMetrics

# clone proj
WORKDIR ${PROJECT_FOLDER}
RUN wget -c https://download.osgeo.org/proj/$PROJ_VERSION.tar.gz
RUN tar -xvzf $PROJ_VERSION.tar.gz
RUN mkdir $PROJ_VERSION/build

# install golang (taken from https://go.dev/doc/install)
WORKDIR /tmp
RUN wget https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}" 

##################################
#       LINUX X64 BUILDER        #
##################################
FROM builder-base AS linux-builder
# install build tools for linux x64 compilation 
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --fix-missing --no-install-recommends build-essential

# install proj dependencies for linux x64
RUN /vcpkg/vcpkg install sqlite3[core,tool] tiff --triplet=x64-linux

# build proj statically for linux x64
WORKDIR ${PROJECT_FOLDER}/${PROJ_VERSION}/build
RUN cmake -DCMAKE_TOOLCHAIN_FILE=/vcpkg/scripts/buildsystems/vcpkg.cmake \
          -DVCPKG_TARGET_TRIPLET=x64-linux \
          -DCMAKE_INSTALL_PREFIX=/usr/local/ \
          -DCMAKE_BUILD_TYPE=Release \
          -DBUILD_APPS=OFF \
          -DBUILD_SHARED_LIBS=OFF \
          -DENABLE_CURL=OFF \
          -DENABLE_TIFF=ON \
          -DBUILD_TESTING=OFF \
          -DEMBED_PROJ_DATA_PATH=OFF \ 
          .. 
RUN cmake --build . --config Release -j $(nproc)
RUN cmake --build . --target install -j $(nproc)

# BUILD_LABEL will force cache invalidation at every build if docker build is run with --build-arg BUILD_LABEL=$(date +%s) 
RUN echo "$BUILD_LABEL"

# clone the source and prepare the build dir
WORKDIR ${PROJECT_FOLDER}/build
COPY . .
RUN mkdir -p ./bin

# BUILD_ID will force cache invalidation at every build if docker build is run with eg --build-arg BUILD_ID=$(date +%s) 
ARG BUILD_ID
RUN echo "build id: $BUILD_ID"

# build the go app for linux x64 statically using cgo
RUN PKG_CONFIG_PATH="/vcpkg/installed/x64-linux/lib/pkgconfig" \
    CGO_ENABLED=1 \
    CGO_LDFLAGS='-L/vcpkg/installed/x64-linux/lib -g -O2 -static -lstdc++ -lsqlite3 -ltiff -lz -ljpeg -llzma -lm' \ 
    go build -o ./bin/gocesiumtiler -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" ./cmd/main.go

# run the unit tests
RUN PROJ_DATA="/usr/local/share/proj" \
    PKG_CONFIG_PATH="/vcpkg/installed/x64-linux/lib/pkgconfig" \
    CGO_ENABLED=1 \
    CGO_LDFLAGS='-L/vcpkg/installed/x64-linux/lib -g -O2 -static -lstdc++ -lsqlite3 -ltiff -lz -ljpeg -llzma -lm' \ 
    go test ./... -v


##################################
#      WINDOWS X64 BUILDER       #
##################################
FROM builder-base AS windows-builder
# install mingw-w64 for cross-compilation to windows
RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y --fix-missing --no-install-recommends mingw-w64 

# set vcpkg env vars to force statically linked build using mingw x664
ENV VCPKG_DEFAULT_TRIPLET=x64-mingw-static
ENV VCPKG_DEFAULT_HOST_TRIPLET=x64-mingw-static

# set cmake default compilers and env vars pointing them to mingw
ENV CMAKE_C_COMPILER=x86_64-w64-mingw32-gcc
ENV CMAKE_CXX_COMPILER=x86_64-w64-mingw32-g++
ENV CMAKE_SYSTEM_NAME=Windows

# install proj dependencies
RUN /vcpkg/vcpkg install sqlite3[core,tool] tiff --triplet=x64-mingw-static

# build proj statically for windows x64
WORKDIR ${PROJECT_FOLDER}/${PROJ_VERSION}/build
RUN cmake -DCMAKE_TOOLCHAIN_FILE=/vcpkg/scripts/buildsystems/vcpkg.cmake \
          -DCMAKE_SYSTEM_NAME=Windows \
          -DVCPKG_TARGET_TRIPLET=x64-mingw-static \
          -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc \
          -DCMAKE_CXX_COMPILER=x86_64-w64-mingw32-g++ \
          -DCMAKE_INSTALL_PREFIX=/usr/local/ \
          -DCMAKE_BUILD_TYPE=Release \
          -DBUILD_APPS=OFF \
          -DBUILD_SHARED_LIBS=OFF \
          -DENABLE_CURL=OFF \
          -DENABLE_TIFF=ON \
          -DBUILD_TESTING=OFF \
          -DEMBED_PROJ_DATA_PATH=OFF \ 
          .. 
RUN cmake --build . --config Release -j $(nproc)
RUN cmake --build . --target install -j $(nproc)

# BUILD_ID will force cache invalidation at every build if docker build is run with eg --build-arg BUILD_ID=$(date +%s) 
RUN echo "build id: $BUILD_ID"

# clone the source and prepare the build dir
WORKDIR ${PROJECT_FOLDER}/build
COPY . .
RUN mkdir -p ./bin

# build the go app for windows x64 statically using cgo
RUN PKG_CONFIG_PATH="/vcpkg/installed/x64-mingw-static/lib/pkgconfig" \
    CC=x86_64-w64-mingw32-gcc \
    CGO_ENABLED=1 \
    CGO_LDFLAGS='-L/vcpkg/installed/x64-mingw-static/lib -g -O2 -static -lstdc++ -lsqlite3 -ltiff -lzlib -ljpeg -llzma -lm' \
    GOOS="windows" \
    GOARCH="amd64" \
    go build -o ./bin/gocesiumtiler.exe -ldflags "-X main.GitCommit=$(git rev-list -1 HEAD)" ./cmd/main.go

##################################
#           Packaging            #
##################################
FROM scratch AS final
ARG PROJECT_FOLDER="/usr/src/gocesiumtiler"
COPY --from=linux-builder /usr/local/share/proj /share
COPY --from=linux-builder ${PROJECT_FOLDER}/build/bin/gocesiumtiler /gocesiumtiler-lin-x64
COPY --from=windows-builder ${PROJECT_FOLDER}/build/bin/gocesiumtiler.exe /gocesiumtiler-win-x64.exe