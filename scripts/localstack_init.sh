#!/bin/bash
#
# This script is intended to be run from within the Docker "localstack" container
# and initializes the default state of the localstack instance
#

# upload default config files to s3 bucket
function init_config_bucket() {
    CONFIG_BUCKET=bcda-local-config
    awslocal s3api create-bucket --bucket $CONFIG_BUCKET
    awslocal s3 sync /etc/config s3://$CONFIG_BUCKET/api
    awslocal s3 sync /etc/config s3://$CONFIG_BUCKET/worker
}

init_config_bucket
