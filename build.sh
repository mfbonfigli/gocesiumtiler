#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

if [[ "${1-}" =~ ^-*h(elp)?$ ]]; then
    echo -e '
\033[1;32mGocesiumtiler build script\033[0m
Usage: ./build.sh

This script automates the building of Gocesiumtiler via Dockerfiles. 
Succesfully compiled binaries will be copied into a ../build subdiractory. 
Note that this directory is wiped out every time the script is invoked.
'
    exit
fi

main() {
    BoldGreen='\033[1;32m'
    Blue='\033[0;34m'
    Cyan='\033[0;36m'

    echo -e "${BoldGreen}Gocesiumtiler build script"
    echo -ne "${Blue} => Removing old build artifacts... "
    rm -rf "./build"
    echo -e "${Blue}done"
    echo -e "${Blue} => Starting dockerized build... "
    build_id="$(date +"%Y%m%d%H%M%S-%N")"
    echo -e "${Blue} => Build id: $build_id"#
    echo -e "${Blue} => Building for arm64..."
    docker buildx build \
        --file Dockerfile.arm64 \
        --platform linux/arm64 \
        --target final \
        --build-arg BUILD_ID=$build_id \
        --output type=local,dest=./build \
        -t gocesiumtiler:build \
        .
    echo -e "${Blue} => Building for x64..."
    docker build -t gocesiumtiler:build --target=final --output ./build --build-arg BUILD_ID=$build_id .
    echo -e "${BoldGreen}=> Build complete, artifacts saved in: $(readlink -f ./build)"
}

main