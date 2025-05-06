#!/usr/bin/env bash
set -eo pipefail

# Get Config
CONFIG=`cat config.json`
# BULK_URL=$(echo $CONFIG | jq -r ".bulk_url")
# TOKEN_URL=$(echo $CONFIG | jq -r ".auth.token_endpoint")
# CLIENT_ID=$(echo $CONFIG | jq -r ".auth.client_id")
# CLIENT_SECRET=$(echo $CONFIG | jq -r ".auth.client_secret")
TESTS=$(echo $CONFIG | jq -r ".required_tests[]")

if [[ "$TOKEN_URL" == "" ]]; then
  echo "
  token_url not set. exiting...
  "
  exit 1
fi

if [[ "$BULK_URL" == "" ]]; then
  echo "
  bulk_url not set. exiting...
  "
  exit 1
fi

if [[ "$CLIENT_ID" == "" ]]; then
  echo "
  client_id not set. exiting...
  "
  exit 1
fi

if [[ "$CLIENT_SECRET" == "" ]]; then
  echo "
  client_secret not set. exiting...
  "
  exit 1
fi

# Get the Token
AUTH_RESP=$(curl -d "" -X POST "$TOKEN_URL" \
	--user "$CLIENT_ID:$CLIENT_SECRET" \
	-H "accept: application/json")

echo "
###
### AUTH RESPONSE: $AUTH_RESPONSE
###
"


TOKEN=$(echo $AUTH_RESP | jq -r ".access_token")

# Create a session
SESSION_RESP=$(curl -d "" -X POST "http://host.docker.internal/api/test_sessions?test_suite_id=bulk_data_v200" \
	-H "accept: application/json")

tmp=${SESSION_RESP#*'"id":"'}
SESSION_ID=${tmp%'","suite_options":'*}

echo "
###
### Session ID: $SESSION_ID
###
"

# Run the tests
TEST_RUN_RESP=$(curl -X POST "http://host.docker.internal/api/test_runs" \
	-H "accept: application/json" \
	-H "Content-Type:application/json" \
	-d '{"test_session_id": "'$SESSION_ID'","test_group_id": "bulk_data_v200-bulk_data_export_tests_v200","inputs": [{"name": "bulk_server_url","value": "'$BULK_URL'"},{"name": "smart_auth_info","value": "{\"access_token\":\"'$TOKEN'\",\"auth_type\":\"backend_services\"}"},{"name": "group_id","value": "all"},{"name": "since_timestamp","value": "2022-10-03T16:03:17-04:00"},{"name": "bulk_timeout","value": "180"}]}')

TEST_RUN_ID=$(echo $TEST_RUN_RESP | jq -r ".id")
echo "
###
### Test Run ID: $TEST_RUN_ID
###
"

# Wait 10 seconds, check to see if the tests are done, repeat (up to 90 seconds)
for i in {1..9}; do
  echo "waiting for job... [$i/9]"
  sleep 10

  TEST_RUN_STATUS_RESPONSE=$(curl -d "" -X GET "http://host.docker.internal/api/test_runs/$TEST_RUN_ID" \
	-H "accept: application/json" )
  TEST_STATUS=$(echo $TEST_RUN_STATUS_RESPONSE | jq -r ".status")

  if [[ "$TEST_STATUS" == "done" ]]; then
    break
  fi

  echo "Job status: $TEST_STATUS"
done

# Exit if the tests still arent done running
if [[ "$TEST_STATUS" != "done" ]]; then
  echo "
  --- Job Timed Out ---
  "

  cd ..
  rm -rf bulk-data-test-kit

  exit 1
fi

# Once it is done, review the results
TEST_RUN_RESULTS=$(curl -d "" -X GET "http://host.docker.internal/api/test_sessions/$SESSION_ID/results" \
	-H "accept: application/json" )

PASS_RESULTS=()
FAIL_RESULTS=()

while read i; do

  # Skip this result if it is for a group, not an individual test
  if [[ $i == *"test_group_id"* ]]; then
    continue 
  fi

  # jq doesn't work because it returns invalid json... so we just find the substring
  tmp=${i#*'"result":"'}
  RESULT=${tmp:0:4}

  tmp=${i#*'"test_id":"'}
  TEST_ID=${tmp%'","test_run_id":'*}

  if [[ "$RESULT" == "pass" ]]; then

    if [[ " ${TESTS[*]} " =~ [[:space:]]${TEST_ID}[[:space:]] ]]; then
      echo "- PASS - $TEST_ID"
      PASS_RESULTS+=($TEST_ID)
    fi
  else
    if [[ " ${TESTS[*]} " =~ [[:space:]]${TEST_ID}[[:space:]] ]]; then
      echo "- $RESULT - $TEST_ID"
      FAIL_RESULTS+=($TEST_ID)
    fi
  fi
done < <(jq -c '.[]' <<< $TEST_RUN_RESULTS)

echo "
SUMMARY:
 - Tests Passed: ${#PASS_RESULTS[@]}
 - Tests Failed: ${#FAIL_RESULTS[@]}
 "

for i in "${FAIL_RESULTS[@]}"; do
   echo "FAIL - $i"
done

if [[ ${#FAIL_RESULTS[@]} == 0 ]]; then
  echo "we gucci"
  exit 0
else
  echo "we failed at least 1"
  exit 1
fi