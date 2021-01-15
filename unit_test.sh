#!/bin/bash
#
# This script is intended to be run from within the Docker "unit_test" container
# The docker-compose file brings forward the env vars: DB
#
set -e
set -o pipefail

timestamp=`date +%Y-%m-%d_%H-%M-%S`
mkdir -p test_results/${timestamp}
mkdir -p test_results/latest

DB_HOST_URL=${DB}?sslmode=disable
TEST_DB_URL=${DB}/bcda_test?sslmode=disable
echo "Running unit tests and placing results/coverage in test_results/${timestamp} on host..."

# Avoid the db/migrations package since it only contains test code
PACKAGES_TO_COVER=$(go list ./... | egrep -v 'test|mock|db/migrations' | tr "\n" "," | sed -e 's/,$//g')
# Leverage the coverpkg flag to allow for coverage of packages even if there are tests outside of its package.
# Useful for packages related to our database interactions.
# Supply the following flag to gotestsum command: -coverpkg ${PACKAGES_TO_COVER}
# NOTE: This flag is a go test flag and should be placed after the -- i.e. same spots as covermode, race, etc.

DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL gotestsum \
    --junitfile test_results/${timestamp}/junit.xml -- \
    -covermode atomic \
    -race ./... \
    -coverprofile test_results/${timestamp}/testcoverage.out 2>&1 | tee test_results/${timestamp}/testresults.out
go tool cover -func test_results/${timestamp}/testcoverage.out > test_results/${timestamp}/testcov_byfunc.out
echo TOTAL COVERAGE:  $(tail -1 test_results/${timestamp}/testcov_byfunc.out | head -1)
go tool cover -html=test_results/${timestamp}/testcoverage.out -o test_results/${timestamp}/testcoverage.html
cp test_results/${timestamp}/* test_results/latest
