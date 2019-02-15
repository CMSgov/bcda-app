#!/usr/bin/env bash

set -eo pipefail

PROJECT_NAME="Beneficiary Claims Data API"

usage() {
    cat <<EOF >&2
Create release notes for a new $PROJECT_NAME release.

Usage: $(basename "$0") [-h] [-p previous_tag] [-n new_tag] [-f release_notes_file]

Options:
  -h    print this help text and exit
  -p    the previous tag (to compare against)
  -n    the new tag
  -f    the file to which the release notes will be written
EOF
}

PREVIOUS_TAG=
NEW_TAG=
RELEASE_NOTES_FILE=
while getopts ":h:p:n:f:" opt; do
    case "$opt" in
        p)
            PREVIOUS_TAG=$OPTARG
            ;;
        n)
            NEW_TAG=$OPTARG
            ;;
        f)
            RELEASE_NOTES_FILE=$OPTARG
            ;;
        h)
            usage
            exit 0
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            exit 1
            ;;
    esac
done

if [ -z "$PREVIOUS_TAG" ] || [ -z "$NEW_TAG" ] || [ -z "$RELEASE_NOTES_FILE" ];
then
  usage
  exit 1
fi

commits=$(git log --pretty=format:"- %s" $PREVIOUS_TAG..$NEW_TAG)
#commits=$(git log --pretty=format:"- %s" $PREVIOUS_TAG..HEAD)

echo "$NEW_TAG - $(date +%Y-%m-%d)" > $RELEASE_NOTES_FILE
echo "================" >> $RELEASE_NOTES_FILE
echo "" >> $RELEASE_NOTES_FILE
echo "$commits" >> $RELEASE_NOTES_FILE
echo "" >> $RELEASE_NOTES_FILE
echo "Release notes created."
