import argparse
import json
import sys
import urllib.request

from argparse import RawTextHelpFormatter


DESCRIPTION = """
Mark a new deployment in New Relic

Example:

    python ./lib/mark_deployment.py \\
        --api_key API_KEY_GOES_HERE \\
        --app_id APP_ID_GOES_HERE \\
        --version VERSION_STRING_GOES_HERE
"""

def main(user, app_id, version, api_key):
    data = {
      "deployment": {
        "revision": version,
        "changelog": ("https://github.com/CMSgov/bcda-app/releases/tag/%s"% version) if version.startswith('r') else '',
        "description": "",
        "user": user
      }
    }
    url = 'https://api.newrelic.com/v2/applications/%s/deployments.json' % app_id
    headers = {
        'Content-Type': 'application/json',
        'X-Api-Key': api_key
    }
    req = urllib.request.Request(
        url, data=json.dumps(data).encode('utf-8'), headers=headers, method='POST')
    resp = urllib.request.urlopen(req)

    if resp.status != 201:
        print("Could not post deployment info to New Relic")
    else:
        print("Successfully marked deployment in New Relic")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description=DESCRIPTION,
        formatter_class=RawTextHelpFormatter
    )

    parser.add_argument(
        '--user', dest='user', type=str, default='jenkins',
        help='Identifies the user marking the deployment in New Relic'
    )

    parser.add_argument(
        '--app_id', dest='app_id', type=str,
        help='The New Relic application ID'
    )

    parser.add_argument(
        '--version', dest='version', type=str,
        help='The version or release number of the deployment'
    )

    parser.add_argument(
        '--api_key', dest='api_key', type=str,
        help='The New Relic API Key used to authenticate'
    )

    args = parser.parse_args()

    if not args.api_key or not args.app_id or not args.version:
        print("Missing required arguments.\n")
        parser.print_help()
        sys.exit(1)

    main(args.user, args.app_id, args.version, args.api_key)
