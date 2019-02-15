#!/usr/bin/env bash

set -eo pipefail

PROJECT_NAME="Beneficiary Claims Data API"
GITHUB_REPO="CMSgov/bcda-app"
ORIGIN="${BCDA_GIT_ORIGIN:-"origin"}"

usage() {
    cat <<EOF >&2
Start a new $PROJECT_NAME release.

Usage: GITHUB_ACCESS_TOKEN=<gh_access_token> $(basename "$0") [-ch] [-t previous-tag new-tag]

Options:
  -h    print this help text and exit
  -t    manually specify tags
EOF
}

MANUAL_TAGS=
while getopts ":chtp" opt; do
    case "$opt" in
        t)
            MANUAL_TAGS=1
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

shift $((OPTIND-1))

if [ $# -lt 2 ] && [ -n "$MANUAL_TAGS" ]
then
  usage
  exit 1
fi

if [ -z "$(echo $(python -V) | grep "Python 3")" ]
then
  echo "Python 3+ is required"
  exit 1
fi

if [ -z "$GITHUB_ACCESS_TOKEN" ]
then
  echo "Please export GITHUB_ACCESS_TOKEN to continue">&2
  exit 1
fi

if [ ! -f ".travis.yml" ]
then
  echo "Must run script in top-level project directory.">&2
  exit 1
fi

# fetch tags before any tag lookups so we have the most up-to-date list
# and generate the correct next release number
git fetch https://${GITHUB_ACCESS_TOKEN}@github.com/CMSgov/bcda-app --tags

if [ -n "$MANUAL_TAGS" ]; then
  PREVTAG="$1"
  NEWTAG="$2"
  PREVRELEASENUM=${PREVTAG//^r/}
  NEWRELEASENUM=${NEWTAG//^r/}
else
  PREVTAG=$(git tag | sort -n | tail -1)
  if [ ! -n "$PREVTAG" ]; then
      PREVRELEASENUM=0
  else
      PREVRELEASENUM=$(git tag | grep '^r[0-9]' | sed 's/^r//' | sort -n | tail -1)
  fi
  NEWRELEASENUM=$(($PREVRELEASENUM + 1))
  PREVTAG="r$PREVRELEASENUM"
  NEWTAG="r$NEWRELEASENUM"
fi

TMPFILE=$(mktemp /tmp/$(basename $0).XXXXXX) || exit 1

bash create_release_notes.sh -p $PREVTAG -n $NEWTAG -f $TMPFILE

git tag -a -m"$PROJECT_NAME release $NEWTAG" -s "$NEWTAG"

python ./ops/github_release.py --release $NEWTAG --release-file $TMPFILE --repo /repos/CMSgov/bcda-app/releases

rm $TMPFILE

echo "Release $NEWTAG created."
echo
