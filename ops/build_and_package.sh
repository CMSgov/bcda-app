#!/bin/bash
#
# This script is intended to be run from within the Docker "package" container
# The version number is a required argument that must be passed to this script.
#
set -e

VERSION=$1

#Prevent ioctl errors - gpg: signing failed: Inappropriate ioctl for device
export GPG_TTY=$(tty)

if [ -z "$VERSION" ]
then
  echo "Please supply version."
  echo "Usage: ./build_and_package.sh <version>"
  exit 1
fi

[  -z "$GPG_RPM_USER" ] && echo "Please enter a Key ID or Username for the GPG Key Signature" && exit 1 || echo "GPG Key user provided"
[  -z "$GPG_PUB_KEY_FILE" ] && echo "Please select a GPG Public Key File" && exit 1 || echo "GPG Public Key File provided"
[  -z "$GPG_SEC_KEY_FILE" ] && echo "Please select a GPG Secret Key File" && exit 1 || echo "GPG Secret Key File provided"
[  -z "$BCDA_GPG_RPM_PASSPHRASE" ] && echo "Please select the Passphrase to sign the RPMs" && exit 1 || echo "GPG Passphrase provided"
[  -z "$GPG_RPM_EMAIL" ] && echo "Please enter the email for the GPG Key Signature" && exit 1 || echo "GPG Key Email provided"

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
fpm -v $VERSION -s dir -t rpm -n bcda bcda=/usr/local/bin/bcda swaggerui=/etc/sv/api _site=/etc/sv/api
cd ../bcdaworker
go clean 
echo "Building bcdaworker..."
go build
echo "Packaging bcdaworker binary into RPM..."
fpm -v $VERSION -s dir -t rpm -n bcdaworker bcdaworker=/usr/local/bin/bcdaworker

#Sign RPMs
WORKER_RPM="bcdaworker-*.rpm"
echo "Importing GPG Key files"
/usr/bin/gpg --batch --import $GPG_PUB_KEY_FILE
/usr/bin/gpg --batch --import $GPG_SEC_KEY_FILE
/usr/bin/rpm --import $GPG_PUB_KEY_FILE

echo "%_signature gpg %_gpg_path $PWD %_gpg_name $GPG_RPM_USER %_gpgbin /usr/bin/gpg" > $PWD/.rpmmacros
echo "allow-loopback-pinentry" > ~/.gnupg/gpg-agent.conf

echo "Signing bcdaworker RPM"
echo $WORKER_RPM
echo $BCDA_GPG_RPM_PASSPHRASE | gpg --batch --yes --passphrase-fd 0 --pinentry-mode loopback --sign $WORKER_RPM

cd ../bcda
BCDA_RPM="bcda-*.rpm"
echo "%_signature gpg %_gpg_path $PWD %_gpg_name $GPG_RPM_USER %_gpgbin /usr/bin/gpg" > $PWD/.rpmmacros
echo "allow-loopback-pinentry" > ~/.gnupg/gpg-agent.conf

echo "Signing bcda RPM"
echo $BCDA_RPM
echo $BCDA_GPG_RPM_PASSPHRASE | gpg --batch --yes --passphrase-fd 0 --pinentry-mode loopback --sign $BCDA_RPM
