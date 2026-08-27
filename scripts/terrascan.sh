#!/usr/bin/env bash

set -e
if [[ -z "$1" ]]; then
    echo "Usage: $0 <directory to scan>" 1>&2
    exit 1
fi
DIR=$1

report_file="terrascan_report.json"

pushd "$DIR"
directory=$(pwd)
files=$(find . -name "*.tf" | xargs)
popd

# bring up terrascan
container=$(docker run --detach -v "${directory}:/temp" --entrypoint "" tenable/terrascan:1.19.9 tail -f /dev/null)

terrascan_cmd="docker exec ${container} /go/bin/terrascan scan -l error --iac-type terraform --iac-version v15 --use-colors f -o json -v"

echo "[" > "${report_file}"
echo "Running terrascan..."

set +e
for f in $files
do
    echo "Scanning $f"
    ${terrascan_cmd} -f "/temp/$f" >> "${report_file}"
    echo "," >> "${report_file}"
done
set -e

# remove last char of a file
truncate -s-2 "${report_file}"

# add closing square bracket
echo "]" >> "${report_file}"

echo "High findings: $(jq '[.[].results.scan_summary.high] | add' < ${report_file})"
echo "Medium findings: $(jq '[.[].results.scan_summary.medium] | add' < ${report_file})"
echo "Low findings: $(jq '[.[].results.scan_summary.low] | add' < ${report_file})"

echo "Please see summary report file for details on how to remediate the findings."

# tear down terrascan
docker stop ${container}
docker rm ${container}
