#!/bin/bash


function exitOnFailure {
    echo "Build failed for ${1}/${2}. Exiting..."
    rm threadfin_privkey.pem
    exit 1
}

function verifySignature {
    # Variables
    binary_file="dist/${1}"
    signature_file="signature.bin"
    original_file="original_binary"
    signature_size=256

    # Extract the signature (assuming the signature is 256 bytes long)
    tail -c $signature_size "$binary_file" > "$signature_file"

    # Extract the original binary content
    head -c -$signature_size "$binary_file" > "$original_file"

    # Verify the signature
    if ! openssl dgst -sha256 -verify threadfin_pubkey.pem -signature "$signature_file" "$original_file"; then
        echo "Signature verification failed for ${1}. Exiting..."
        exit 1
    fi

    # Clean up
    rm "$signature_file" "$original_file"
}

os_list=("darwin" "freebsd" "linux" "windows")
arch_list=("amd64" "arm64")

echo "$PRIVATE_KEY" > threadfin_privkey.pem

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
            if ! go build -o "dist/${bin_string}" -tags beta; then
                exitOnFailure "$os" "$arch"
            fi
        else
            if ! go build -o "dist/${bin_string}"; then 
                exitOnFailure "$os" "$arch"
            fi
        fi

        echo "Signing the binary"
        openssl dgst -sha256 -sign threadfin_privkey.pem -out signature.bin "dist/${bin_string}"
        cat signature.bin >> "dist/${bin_string}"

        echo "Verify signature"
        verifySignature "$bin_string"

        echo "Calculating sha256 for ${bin_string}"
        sha256sum "dist/${bin_string}" > "dist/${sha_string}.sha256"
    done
done

rm threadfin_privkey.pem