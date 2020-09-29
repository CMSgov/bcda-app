#!/bin/bash

CMS_IDs=("A9996" "E9996" "V996")
set -e
function cleanup() {
        for CMS_ID in "${CMS_IDs[@]}"
        do 
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM encryption_keys WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM secrets WHERE system_id IN (SELECT id FROM systems WHERE group_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM systems WHERE group_id = '"'"${CMS_ID}"'"'"'
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM groups WHERE group_id = '"'"${CMS_ID}"'"'"'
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM jobs WHERE aco_id IN (SELECT uuid FROM acos where cms_id = '"'"${CMS_ID}"'"')"'
                docker-compose exec db sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM acos WHERE cms_id = '"'"${CMS_ID}"'"'"'
        done
}
trap cleanup EXIT

docker-compose stop api
SSAS_URL="http://ssas:3004" SSAS_PUBLIC_URL="http://ssas:3003" BCDA_AUTH_PROVIDER=ssas BCDA_SSAS_CLIENT_ID=$BCDA_SSAS_CLIENT_ID BCDA_SSAS_SECRET=$BCDA_SSAS_SECRET SSAS_ADMIN_CLIENT_ID=$BCDA_SSAS_CLIENT_ID SSAS_ADMIN_CLIENT_SECRET=$BCDA_SSAS_SECRET DEBUG=true docker-compose up -d api ssas

echo "waiting for API to rebuild with SSAS as auth provider"
sleep 30

for CMS_ID in "${CMS_IDs[@]}"
do
        ACO_ID=$(docker-compose exec -e CMS_ID=${CMS_ID} api sh -c 'tmp/bcda create-aco --name "Smoke Test ACO" --cms-id ${CMS_ID}' | tail -n1 | tr -d '\r')
        SAVE_KEY_RESULT=$(docker-compose exec -e CMS_ID=${CMS_ID} api sh -c 'tmp/bcda save-public-key --cms-id ${CMS_ID} --key-file "../shared_files/ATO_public.pem"' | tail -n1)
        GROUP_ID=$(docker-compose exec -e CMS_ID=${CMS_ID} api sh -c 'tmp/bcda create-group --id ${CMS_ID} --name "Smoke Test Group" --aco-id ${CMS_ID}' | tail -n1 | tr -d '\r')
        CREDS=($(docker-compose exec -e CMS_ID=${CMS_ID} api sh -c 'tmp/bcda generate-client-credentials --cms-id ${CMS_ID}' | tail -n2 | tr -d '\r'))
        CLIENT_ID=${CREDS[0]}
        CLIENT_SECRET=${CREDS[1]}

        testFile=bulk_data_requests_lite.sh
        if [ ${CMS_ID} = "A9996" ]; then
                testFile=bulk_data_requests.sh
        fi
        CLIENT_ID=${CLIENT_ID} CLIENT_SECRET=${CLIENT_SECRET} docker-compose -f docker-compose.test.yml run --rm -w /go/src/github.com/CMSgov/bcda-app/test/smoke_test tests sh ${testFile} &
done

wait
docker-compose stop api ssas

echo "waiting for API to rebuild with alpha as auth provider"
sleep 30
docker-compose up -d api
