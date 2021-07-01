#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e

pids=()

echo "Running EOB" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=ExplanationOfBenefit && \
echo "Running Patient" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient && \
echo "Running Coverage" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage && \
echo "Running Patient All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient &
pids+=($!)

echo "Running Patient, Coverage, EOB (explicitly)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage,ExplanationOfBenefit && \
echo "Running Patient, Coverage" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage && \
echo "Running Patient, EOB" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,ExplanationOfBenefit && \
echo "Running Coverage, EOB" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage,ExplanationOfBenefit &
pids+=($!)

echo "Running EOB v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=ExplanationOfBenefit -apiVersion=v2 && \
echo "Running Patient v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient -apiVersion=v2 && \
echo "Running Coverage v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage -apiVersion=v2 && \
echo "Running Patient v2 All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -apiVersion=v2 &
pids+=($!)

echo "Running Patient v2 Patient, Coverage, EOB (explicitly)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage,ExplanationOfBenefit -apiVersion=v2 && \
echo "Running Patient, Coverage v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage -apiVersion=v2 && \
echo "Running Patient, EOB v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,ExplanationOfBenefit -apiVersion=v2 && \
echo "Running Coverage, EOB v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage,ExplanationOfBenefit -apiVersion=v2 &
pids+=($!)

echo "Running Group All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/all && \
echo "Running Group Runout (EOB resource)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/runout -resourceType=ExplanationOfBenefit && \
echo "Running Group All v2" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/all -apiVersion=v2 && \
echo "Running Group Runout v2 (EOB resource)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/runout -resourceType=ExplanationOfBenefit -apiVersion=v2 &
pids+=($!)

for pid in "${pids[@]}"; do
   wait "$pid"
done
