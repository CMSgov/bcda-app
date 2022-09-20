#!/bin/bash

CMS_IDs=("A9996")
set -e
function cleanup() {
        for CMS_ID in "${CMS_IDs[@]}"
        do 
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM encryption_keys WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM secrets WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM systems WHERE group_id = '"'"${CMS_ID}"'"'"'
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM groups WHERE group_id = '"'"${CMS_ID}"'"'"'
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM jobs WHERE aco_id IN (SELECT uuid FROM acos where cms_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec -T db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM acos WHERE cms_id = '"'"${CMS_ID}"'"'"'
        done
}
trap cleanup EXIT

for CMS_ID in "${CMS_IDs[@]}"
do
        ACO_ID=$(docker-compose exec -T -e CMS_ID=${CMS_ID} api sh -c 'bcda create-aco --name "Smoke Test ACO" --cms-id ${CMS_ID}' | tail -n1 | tr -d '\r')
        GROUP_ID=$(docker-compose exec -T -e CMS_ID=${CMS_ID} api sh -c 'bcda create-group --id ${CMS_ID} --name "Smoke Test Group" --aco-id ${CMS_ID}' | tail -n1 | tr -d '\r')
        CREDS=($(docker-compose exec -T -e CMS_ID=${CMS_ID} api sh -c 'bcda generate-client-credentials --cms-id ${CMS_ID}' | tail -n2 | tr -d '\r'))
        CLIENT_ID=${CREDS[0]}
        CLIENT_SECRET=${CREDS[1]}
        ENV=$1

        docker-compose -f docker-compose.test.yml build postman_test
        docker-compose -f docker-compose.test.yml run --rm postman_test test/postman_test/BCDA_Postman_Smoke_Tests.postman_collection.json \
	-e test/postman_test/${ENV}.postman_environment.json --global-var clientId=${CLIENT_ID} --global-var clientSecret=${CLIENT_SECRET}
done

