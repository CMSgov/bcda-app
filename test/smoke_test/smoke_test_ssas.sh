#!/bin/bash
pwd
cd ../..
pwd
ACO_ID=$('tmp/bcda create-aco --name "Smoke Test ACO" --cms-id A9996' | tail -n1 | tr -d '\r')
echo "ACO: $ACO_ID"
