#!/usr/bin/env bash

VERSION=$1

if [[ -z $VERSION ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi


for i in $(ls release); do
    fileName="devkit-${i}-${VERSION}.tar.gz"

    tar -cvf "./release/${fileName}" -C "./release/${i}/" devkit
done
