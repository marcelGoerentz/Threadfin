#!/bin/bash

os_list=("darwin" "freebsd" "linux" "windows")
arch_list=("amd64" "arm64")

for os in "${os_list[@]}"; do
    export GOOS=$os
    for arch in "${arch_list[@]}"; do 
        export GOARCH=$arch
        bin_string="Threadfin"
        if [ "$1" = "beta" ]; then
            bin_string="${bin_string}_beta"
        fi
        bin_string="${bin_string}_${os}_${arch}"
        sha_string="${bin_string}"
        if [ "$os" = "windows" ]; then 
            bin_string="${bin_string}.exe"
        fi
        echo "Building ${bin_string}"
        if [ "$1" = "beta" ]; then
            go build -o "dist/${bin_string}" -tags beta
        else
            go build -o "dist/${bin_string}"
        fi
        echo "Calculating sha256 for ${bin_string}"
        sha256sum "dist/${bin_string}" > "dist/${sha_string}.sha256"
    done
done