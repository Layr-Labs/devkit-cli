#!/usr/bin/env bash

REF=$1

versionFile=$(cat VERSION | tr -d '[:space:]')
echo "Using version '${versionFile}'"
if [[ $REF == refs/tags/* ]]; then
    # check if versionFile equals the tag.
    if [[ $versionFile != "${REF#refs/tags/}" ]]; then
      echo "Version in VERSION file does not match the tag"
      exit 1
    fi
    echo "Version correctly matches tag"
else
    # if this isnt a release, add the commit hash to the end of the version
    v=$(git rev-parse --short HEAD)
    updatedVersion="${versionFile}+${v}"
    echo "Updated version to '${updatedVersion}'"
    echo -n $updatedVersion > VERSION
fi
