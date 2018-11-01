#!/bin/bash
#
# This script is intended to be run from within the Docker "package" container
# The version number is a required argument that must be passed to this script.
#
set -e

VERSION=$1

if [ -z "$VERSION" ]
then
  echo "Please supply version."
  echo "Usage: ./build_and_package.sh <version>"
  exit 1
fi

if [ ! -f ../bcda/swaggerui/swagger.json ]
then
  echo "Swagger doc generation must be completed prior to creating package."
  exit 1
fi

cd ../bcda
go clean
echo "Building bcda binary..." 
go build -ldflags "-X main.version=$VERSION"
echo "Packaging bcda binary into RPM..."
fpm -v $VERSION -s dir -t rpm -n bcda bcda=/usr/local/bcda/bcda swaggerui=/usr/local/bcda
cd ../bcdaworker
go clean 
echo "Building bcdaworker..."
go build
echo "Packaging bcdaworker binary into RPM..."
fpm -v $VERSION -s dir -t rpm -n bcdaworker bcdaworker=/usr/local/bcda/bcdaworker

