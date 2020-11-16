#!/bin/bash

# Script used to seed test data within the postgres docker container
PGPASSWORD=${POSTGRES_PASSWORD}  pg_restore -C -d postgres -v -p 5432 -U postgres --clean --exit-on-error --if-exists "/docker-entrypoint-initdb.d/dump.pgdata"
