#!/usr/bin/env bash

set -eo pipefail

if [ -z "$BASE_URL" ]
then
    echo "Please export BASE_URL to continue"
    exit 1
fi

if [ -z "$CLIENT_ID" ]
then
    echo "Please export CLIENT_ID to continue"
    exit 1
fi

if [ -z "$SECRET" ]
then
    echo "Please export SECRET to continue"
    exit 1
fi

NODE_OPTIONS="--max-old-space-size=8192" \
BASE_URL=${BASE_URL} \
CLIENT_ID=${CLIENT_ID} \
CLIENT_SECRET=${SECRET} \
node index.js --pattern "testSuite/**/!(authorization.test.js)"