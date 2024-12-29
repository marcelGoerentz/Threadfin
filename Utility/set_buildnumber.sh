#!/bin/bash

if [ -z "$1" ]; then
    echo "No argument provided"
    exit 1
fi

new_build=$(($1 + 1))

echo "New buildnumber is: $new_build"

# Extract the version number
version=$(grep 'const Version' "threadfin.go" | sed 's/.*Version = "\(.*\)"/\1/')

# Split the version into parts
IFS='.' read -r -a version_parts <<< "$version"

# Update the version parts
major=${version_parts[0]}
minor=${version_parts[1]}
patch=${version_parts[2]}
new_version="$major.$minor.$patch.$new_build"

echo "New version is: $new_version"

# Update the version in the file
sed -i "s/const Version = \".*\"/const Version = \"$new_version\"/" threadfin.go

# Export the new build number to the GitHub environment
echo "NEW_BUILD=$new_build" >> $GITHUB_ENV

# Export the new Version to the GitHub environment
echo "NEW_VERSION=$new_version" >> $GITHUB_ENV
