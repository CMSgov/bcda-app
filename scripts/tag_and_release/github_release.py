import argparse
import json
import os
import sys
import urllib.request
from urllib.parse import urlparse


def safe_path(path):
    resolved = os.path.realpath(path)
    base_dir = os.path.realpath(os.getcwd())
    if resolved != base_dir and not resolved.startswith(base_dir + os.sep):
        raise ValueError(f"path {path!r} is outside the allowed directory")
    return resolved

def main(release, release_file, repo):
    access_token = os.environ['GITHUB_ACCESS_TOKEN']

    resp = None
    with open(safe_path(release_file), 'r') as f:
        data = {
            "tag_name": release,
            "name": release,
            "body": f.read(),
            "draft": False,
            "prerelease": False
        }

        url = "https://api.github.com" + repo
        headers = {
            "Authorization": "Bearer %s" % access_token
        }

        if urlparse(url).hostname in 'https://api.github.com':
            req = urllib.request.Request(
                url,
                data=json.dumps(data).encode('utf-8'),
                headers=headers,
                method='POST'
            )
            resp = urllib.request.urlopen(req)


    if not resp or resp.status != 201:
        print("Could not create release: %s" % release)
        sys.exit(1)
    else:
        print("Successfully created release: %s" % release)

def verify_repo(repo):
    if not repo.startswith('/repos/CMSgov/bcda'):
        raise argparse.ArgumentTypeError(f"non-bcda repo '{repo}' passed as argument")

def verify_release(release):
    if not release.startswith('r'):
        raise argparse.ArgumentTypeError(f"invalid release tag '{release}' passed as argument")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument(
        '--release', dest='release', type=verify_release,
        help='The version tag/identifier for the release'
    )

    parser.add_argument(
        '--release-file', dest='release_file', type=str,
        help='Path to file with body of release notes'
    )
 
    parser.add_argument(
        '--repo', dest='repo', type=verify_repo,
        help='The repository of the release (i.e., /repos/CMSgov/bcda-app/releases)'
    )

    args = parser.parse_args()

    main(args.release, args.release_file, args.repo)
