#!/bin/bash
#
# This script is intended to be run from within the Docker "localstack" container
# and initializes the default state of the localstack instance
#

# upload default config files to s3 bucket
function upload_config_s3() {
    service=$1
    CONFIG_BUCKET=bcda-local-$service-config
    awslocal s3api create-bucket --bucket $CONFIG_BUCKET
    awslocal s3 sync /etc/config s3://$CONFIG_BUCKET
}

upload_config_s3 "api"
upload_config_s3 "worker"
