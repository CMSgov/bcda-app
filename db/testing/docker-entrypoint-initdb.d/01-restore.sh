#!/bin/bash

# Script used to seed test data within the postgres docker container

# Seeding the database using pg_restore replaces the legacy flow of usql/import-synthetic-cclf-package. That process is captured below for tracking purposes.
# usql $DB_HOST_URL -c 'create database bcda_test;'
# usql $TEST_DB_URL -f db/api.sql
# usql $TEST_DB_URL -f db/fixtures.sql
# usql $TEST_DB_URL -f db/worker.sql
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda sql-migrate
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize dev --environment unit-test
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize small --environment unit-test
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize medium --environment unit-test
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize large --environment unit-test
# usql $TEST_DB_URL -c "update cclf_files set timestamp='2020-02-01';"
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize dev --environment unit-test-new-beneficiaries
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize small --environment unit-test-new-beneficiaries
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize medium --environment unit-test-new-beneficiaries
# DATABASE_URL=$TEST_DB_URL QUEUE_DATABASE_URL=$TEST_DB_URL go run github.com/CMSgov/bcda-app/bcda import-synthetic-cclf-package --acoSize large --environment unit-test-new-beneficiaries

PGPASSWORD=${POSTGRES_PASSWORD}  pg_restore -C -d postgres -v -p 5432 -U postgres --clean --exit-on-error --if-exists "/docker-entrypoint-initdb.d/dump.pgdata"
