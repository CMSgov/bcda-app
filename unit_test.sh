#!/bin/bash

#
# This script is intended to be run from within the Docker "unit_test" container
# The docker-compose file brings forward the env vars: DB_HOST_URL and TEST_DB_URL
#

echo "Running linter..."
golangci-lint run

echo "Setting up test DB (bcda_test)..."
echo "DB_HOST_URL is $DB_HOST_URL"
echo "TEST_DB_URL is $TEST_DB_URL"
usql $DB_HOST_URL -c 'drop database if exists bcda_test;'
usql $DB_HOST_URL -c 'create database bcda_test;'
usql $TEST_DB_URL -f db/api.sql
usql $TEST_DB_URL -f db/fixtures.sql

echo "Running unit tests..."
DATABASE_URL=$TEST_DB_URL go test -v -race ./...

echo "Cleaning up test DB (bcda_test)..."
usql $DB_HOST_URL -c 'drop database bcda_test;'
