import argparse
import json
import os
import sys
import urllib.request


def main(release, release_file):
    access_token = os.environ['GITHUB_ACCESS_TOKEN']

    with open(release_file, 'r') as f:
        data = {
            "tag_name": release,
            "name": release,
            "body": f.read(),
            "draft": False,
            "prerelease": False
        }

        base_url = "https://api.github.com"
        path = "/repos/CMSgov/bcda-app/releases"
        headers = {
            "Authorization": "token %s" % access_token
        }

        req = urllib.request.Request(
            base_url + path, data=json.dumps(data).encode('utf-8'),
            headers=headers,
            method='POST'
        )
        resp = urllib.request.urlopen(req)

    if resp.status != 201:
        print("Could not create release: %s" % release)
        sys.exit(1)
    else:
        print("Successfully created release: %s" % release)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument(
        '--release', dest='release', type=str,
        help='The version tag/identifier for the release'
    )

    parser.add_argument(
        '--release-file', dest='release_file', type=str,
        help='Path to file with body of release notes'
    )

    args = parser.parse_args()

    main(args.release, args.release_file)
