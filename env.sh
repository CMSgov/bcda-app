#!/bin/bash	
#	
# Edit and source this script to add any environment 	
# variables for local development	
#	
# This file is included in .gitignore and should not	
# be committed as it may contain secrets	
#	
export BCDA_AUTH_PROVIDER=okta	
export OKTA_OAUTH_SERVER_ID="<serverID>"	
export OKTA_API_KEY="<apiKey>"	
export OKTA_CLIENT_SECRET="<clientSecret>"