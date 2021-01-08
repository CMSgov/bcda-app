#!/bin/bash

# Script used to seed test data within the postgres docker container
PGPASSWORD=${POSTGRES_PASSWORD}  pg_restore -C -d postgres -v -p 5432 -U postgres --clean --exit-on-error --if-exists "/docker-entrypoint-initdb.d/dump.pgdata"
# Update the suppression date to guarantee suppression entries satisfy the date requirements.
# See postgres#GetSuppressedMBIs for more information
PGPASSWORD=$POSTGRES_PASSWORD psql $POSTGRES_DB postgres -c 'UPDATE suppressions SET effective_date = now() WHERE effective_date = (SELECT max(effective_date) FROM suppressions);'
