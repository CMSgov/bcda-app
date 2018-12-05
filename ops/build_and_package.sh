#!/bin/bash
#
# This script is intended to be run from within the Docker "package" container
# The version number is a required argument that must be passed to this script.
#
set -e

VERSION=$1
GPG_RPM_USER="Beneficiary Claims Data API"
GPG_RPM_EMAIL="bcda-ops.group@adhocteam.us"
GPG_PUB_KEY_FILE="../ops/RPM-GPG-KEY-bcda"
GPG_SEC_KEY_FILE="../ops/RPM-GPG-KEY-SECRET-bcda"
WORKER_RPM="bcdaworker-*.rpm"
BCDA_RPM="bcda-*.rpm"

#Prevent ioctl errors - gpg: signing failed: Inappropriate ioctl for device
export GPG_TTY=$(tty)

if [ -z "$VERSION" ]
then
  echo "Please supply version."
  echo "Usage: ./build_and_package.sh <version>"
  exit 1
fi

if [ ! -f ../bcda/swaggerui/swagger.json ]
then
  echo "Swagger doc generation must be completed prior to creating package."
  exit 1
fi

cd ../bcda
go clean
echo "Building bcda binary..." 
go build -ldflags "-X main.version=$VERSION"
echo "Packaging bcda binary into RPM..."
fpm -v $VERSION -s dir -t rpm -n bcda bcda=/usr/local/bin/bcda swaggerui=/etc/sv/api
cd ../bcdaworker
go clean 
echo "Building bcdaworker..."
go build
echo "Packaging bcdaworker binary into RPM..."
fpm -v $VERSION -s dir -t rpm -n bcdaworker bcdaworker=/usr/local/bin/bcdaworker

#Sign RPMs
echo "Importing GPG Key files"
/usr/bin/gpg --batch --import $GPG_PUB_KEY_FILE
/usr/bin/gpg --batch --import $GPG_SEC_KEY_FILE
/usr/bin/rpm --import $GPG_PUB_KEY_FILE

echo "%_signature gpg %_gpg_path $PWD %_gpg_name $GPG_RPM_USER %_gpgbin /usr/bin/gpg" > $PWD/.rpmmacros
echo "allow-loopback-pinentry" > ~/.gnupg/gpg-agent.conf

echo "Signing bcdaworker RPM"
echo $BCDA_GPG_RPM_PASSPHRASE | gpg --batch --yes --passphrase-fd 0 -v --pinentry-mode loopback --sign $WORKER_RPM

cd ../bcda
echo "%_signature gpg %_gpg_path $PWD %_gpg_name $GPG_RPM_USER %_gpgbin /usr/bin/gpg" > $PWD/.rpmmacros
echo "allow-loopback-pinentry" > ~/.gnupg/gpg-agent.conf

echo "Signing bcda RPM"
echo $BCDA_GPG_RPM_PASSPHRASE | gpg --batch --yes --passphrase-fd 0 -v --pinentry-mode loopback --sign $BCDA_RPM
