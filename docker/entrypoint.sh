#!/bin/sh
#
# This script is intended to be run from within the service Docker container
# and perform basic configuration before starting the service.
#

set -e

bootstrap_config() {
  # Sync the config aws bucket
  aws s3 sync "s3://$CONFIG_BUCKET/$APP_NAME" /etc/sv/$APP_NAME/env/
}

if ! [[ -n "$SKIP_BOOTSTRAP" ]]; then
  echo "Bootstrapping config from S3"
  bootstrap_config
fi

echo "Starting main process"
exec "$@"
