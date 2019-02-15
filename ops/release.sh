#!/usr/bin/env bash

set -eo pipefail

PROJECT_NAME="Beneficiary Claims Data API"

usage() {
    cat <<EOF >&2
Start a new $PROJECT_NAME release.

Usage: GITHUB_REPO_PATH=<gh_repo> GITHUB_ACCESS_TOKEN=<gh_access_token> $(basename "$0") [-ch] [-t previous-tag new-tag]

Optionally, GITHUB_USER, GITHUB_EMAIL, and GITHUB_GPG_KEY_FILE environment variables can be set prior to running this script, to identify and verify who is creating the release.  This is primarily necessary when the release process is run from a Docker container (i.e., from Jenkins).

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

if [ -z "$GITHUB_REPO_PATH" ]
then
  echo "Please export GITHUB_REPO_PATH to continue (i.e., /CMSgov/bcda-app">&2
  exit 1
fi

# initialize git configuration if env vars are set
if [ ! -z "$GITHUB_USER" ] && [ ! -z "$GITHUB_EMAIL" ] && [ ! -z "$GITHUB_GPG_KEY_FILE" ]
then
  git config user.name "$GITHUB_USER"
  git config user.email "$GITHUB_EMAIL"
  gpg --import $GITHUB_GPG_KEY_FILE
fi

# fetch tags before any tag lookups so we have the most up-to-date list
# and generate the correct next release number
git fetch https://${GITHUB_ACCESS_TOKEN}@github.com$GITHUB_REPO_PATH --tags

if [ -n "$MANUAL_TAGS" ]; then
  PREVTAG="$1"
  NEWTAG="$2"
  PREVRELEASENUM=${PREVTAG//^r/}
  NEWRELEASENUM=${NEWTAG//^r/}
else
  PREVTAG=$(git tag | sort -n | tail -1)
  if [ ! -n "$PREVTAG" ]; then
      PREVRELEASENUM=
  else
      PREVRELEASENUM=$(git tag | grep '^r[0-9]' | sed 's/^r//' | sort -n | tail -1)
  fi
  NEWRELEASENUM=$(($PREVRELEASENUM + 1))
  PREVTAG="r$PREVRELEASENUM"
  NEWTAG="r$NEWRELEASENUM"
fi

TMPFILE=$(mktemp /tmp/$(basename $0).XXXXXX) || exit 1

if [ -z $PREVRELEASENUM ]
then
  commits=$(git log --pretty=format:"- %s" $PREVTAG..HEAD)
else
  commits=$(git log --pretty=format:"- %s" HEAD)
fi

echo "$NEWTAG - $(date +%Y-%m-%d)" > $TMPFILE
echo "================" >> $TMPFILE
echo "" >> $TMPFILE
echo "$commits" >> $TMPFILE
echo "" >> $TMPFILE

git tag -a -m"$PROJECT_NAME release $NEWTAG" -s "$NEWTAG"

RELEASE_PATH="/repos$GITHUB_REPO_PATH/releases"
python github_release.py --release $NEWTAG --release-file $TMPFILE --repo $RELEASE_PATH

rm $TMPFILE

echo "Release $NEWTAG created."
echo
