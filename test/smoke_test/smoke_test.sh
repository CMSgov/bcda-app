#!/bin/bash

set -e
function cleanup {
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM jobs WHERE aco_id = '"'"$ACO_ID"'"'"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM users WHERE aco_id = '"'"$ACO_ID"'"'"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM acos WHERE cms_id = '"'"'A9996'"'"'"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM encryption_keys WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"'smoke-test-group'"'"')"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM secrets WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"'smoke-test-group'"'"')"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM systems WHERE group_id = '"'"'smoke-test-group'"'"'"'
        docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM groups WHERE group_id = '"'"'smoke-test-group'"'"'"'
}
trap cleanup EXIT

docker-compose stop api
SSAS_URL="http://ssas:3004" SSAS_PUBLIC_URL="http://ssas:3003" BCDA_AUTH_PROVIDER=ssas BCDA_SSAS_CLIENT_ID=$BCDA_SSAS_CLIENT_ID BCDA_SSAS_SECRET=$BCDA_SSAS_SECRET SSAS_ADMIN_CLIENT_ID=$BCDA_SSAS_CLIENT_ID SSAS_ADMIN_CLIENT_SECRET=$BCDA_SSAS_SECRET DEBUG=true docker-compose up -d api ssas

echo "waiting for API to rebuild with SSAS as auth provider"
sleep 30

ACO_ID=$(docker-compose exec api sh -c 'tmp/bcda create-aco --name "Smoke Test ACO" --cms-id A9996' | tail -n1 | tr -d '\r')
USER_ID=$(docker-compose exec api sh -c 'tmp/bcda create-user --name "SSAS Smoke Test User" --email ssassmoketest@example.com --aco-id '$ACO_ID | tail -n1)
SAVE_KEY_RESULT=$(docker-compose exec api sh -c 'tmp/bcda save-public-key --cms-id A9996 --key-file "../shared_files/ATO_public.pem"' | tail -n1)
GROUP_ID=$(docker-compose exec api sh -c 'tmp/bcda create-group --id "smoke-test-group" --name "Smoke Test Group" --aco-id A9996' | tail -n1 | tr -d '\r')
CREDS=($(docker-compose exec api sh -c 'tmp/bcda generate-client-credentials --cms-id A9996' | tail -n2 | tr -d '\r'))
CLIENT_ID=${CREDS[0]}
CLIENT_SECRET=${CREDS[1]}

CLIENT_ID=$CLIENT_ID CLIENT_SECRET=$CLIENT_SECRET docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/smoke_test tests sh bulk_data_requests.sh
docker-compose stop api ssas

echo "waiting for API to rebuild with alpha as auth provider"
sleep 30
docker-compose up -d api