#!/bin/bash
ACO_ID=$(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda create-aco --name "Smoke Test ACO" --cms-id A9996' | tail -n1 | tr -d '\r')
echo "ACO: $ACO_ID"
USER_ID=$(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda create-user --name "SSAS Smoke Test User" --email ssassmoketest@example.com --aco-id '$ACO_ID | tail -n1)
echo "User: $USER_ID"
SAVE_KEY_RESULT=$(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda save-public-key --cms-id A9996 --key-file "../shared_files/ATO_public.pem"' | tail -n1)
echo "Save key: $SAVE_KEY_RESULT"
GROUP_ID=$(docker-compose run --rm api sh -c 'tmp/bcda create-group --id "smoke-test-group" --name "Smoke Test Group" --aco-id A9996' | tail -n1 | tr -d '\r')
echo "Group: $GROUP_ID"
CREDS=($(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda generate-client-credentials --cms-id A9996' | tail -n2 | tr -d '\r'))
CLIENT_ID=${CREDS[0]}
CLIENT_SECRET=${CREDS[1]}
echo "Client ID: $CLIENT_ID"
echo "Client secret: $CLIENT_SECRET"

ATO_PRIVATE_KEY_FILE=../../shared_files/ATO_private.pem go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient
ATO_PRIVATE_KEY_FILE=../../shared_files/ATO_private.pem go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Coverage
ATO_PRIVATE_KEY_FILE=../../shared_files/ATO_private.pem go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=ExplanationOfBenefit

docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM jobs WHERE aco_id = '"'"$ACO_ID"'"'"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM users WHERE aco_id = '"'"$ACO_ID"'"'"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM acos WHERE cms_id = '"'"'A9996'"'"'"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM encryption_keys WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"'smoke-test-group'"'"')"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM secrets WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"'smoke-test-group'"'"')"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM systems WHERE group_id = '"'"'smoke-test-group'"'"'"'
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM groups WHERE group_id = '"'"'smoke-test-group'"'"'"'