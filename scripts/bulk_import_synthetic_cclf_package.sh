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

# The project root contains a migrations directory named `db`.  V3 of jackc/pgx has
# a bug that causes the specified DB connection to be ignored when CWD has a directory
# like this.  To get around this we must execute the `bcda` binary from a directory
# that does not have a `db` directory.  https://github.com/jackc/pgx/issues/661
cd bcda
for size in "${@}"
do
    bcda import-synthetic-cclf-package --acoSize=$size --environment=$environment --fileType=$filetype
done
