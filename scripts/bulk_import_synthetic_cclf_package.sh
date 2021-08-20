#!/bin/sh

# script/bulk_import_synthetic_cclf_package: load multiple CCLF packages
# eg ./bulk_import_synthetic_cclf_package.sh <environment> <fileType> <acoSize1> <acoSize2> ...

set -e

cd "$(dirname "$0")/.."

if [ $# -lt 3 ]; then
    echo >&2 "an environment, fileType and at least 1 acoSize must be provided"
    exit 1
fi

environment=$1
filetype=$2
shift;shift

# For some reason, the `bcda` command only works if executed within the bcda directory
cd bcda
for size in "${@}"
do
    bcda import-synthetic-cclf-package --acoSize=$size --environment=$environment --fileType=$filetype
done
