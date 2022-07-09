#!/bin/bash

# Get VERSIONSTRING
VERSIONSTRING="$(git describe --tags --exact-match || git rev-parse --short HEAD)"
echo "Building version ${VERSIONSTRING} of Plantmonitor ..."

# Compile and link statically and remove debug symbols
# Add version string to source
CGO_ENABLED=0 GOOS=linux \
go build -a -ldflags "-s -w -extldflags '-static' -X main.versionString=${VERSIONSTRING}"

