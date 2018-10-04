#!/bin/bash
#
# This script is intended to be run from within the Docker "unit_test" container
# The docker-compose file brings forward the env vars: DB
#
set -e
set -o pipefail

echo "Running linter..."
golangci-lint run

echo "Setting up test DB (bcda_test)..."
DB_HOST_URL=${DB}?sslmode=disable
TEST_DB_URL=${DB}/bcda_test?sslmode=disable
echo "DB_HOST_URL is $DB_HOST_URL"
echo "TEST_DB_URL is $TEST_DB_URL"
usql $DB_HOST_URL -c 'drop database if exists bcda_test;'
usql $DB_HOST_URL -c 'create database bcda_test;'
usql $TEST_DB_URL -f db/api.sql
usql $TEST_DB_URL -f db/fixtures.sql

timestamp=`date +%Y-%m-%d_%H-%M-%S`
echo "Running unit tests and placing results/coverage in test_results/${timestamp} on host..."
mkdir -p test_results/${timestamp}
mkdir -p test_results/latest
DATABASE_URL=$TEST_DB_URL go test -v -race ./... -coverprofile test_results/${timestamp}/testcoverage.out 2>&1 | tee test_results/${timestamp}/testresults.out
go tool cover -func test_results/${timestamp}/testcoverage.out > test_results/${timestamp}/testcov_byfunc.out
echo TOTAL COVERAGE:  $(tail -1 test_results/${timestamp}/testcov_byfunc.out | head -1)
go tool cover -html=test_results/${timestamp}/testcoverage.out -o test_results/${timestamp}/testcoverage.html
cp test_results/${timestamp}/* test_results/latest
echo "Cleaning up test DB (bcda_test)..."
usql $DB_HOST_URL -c 'drop database bcda_test;'
