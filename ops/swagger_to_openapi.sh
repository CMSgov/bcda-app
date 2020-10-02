#!/bin/sh
#
# This script is intended to be run from within the Docker "openapi_converter" container
#
set -e
set -o pipefail

java -jar -DswaggerUrl=swagger.yaml /swagger-converter/jetty-runner.jar /swagger-converter/server.war &
sleep 30
curl -X POST -d "@swaggerui/swagger.json" localhost:8080/api/convert -H "Content-Type: application/json" -o temp.json
cat temp.json | jq > swaggerui/openapi.json

# Remove openapi.json file if it is invalid
# Note that build_and_package.sh script validates that openapi.json exists; so RPM packaging will fail if there is an invalid openapi.json
grep "Beneficiary Claims Data API" swaggerui/openapi.json || rm swaggerui/openapi.json
