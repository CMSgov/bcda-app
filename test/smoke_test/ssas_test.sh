#GROUP_ID=$(docker-compose run --rm api sh -c 'tmp/bcda create-group --id="5a0b3-7357" --name="Smoke Test Group"' | tail -n1)
ACO_ID=$(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda create-aco --name="Smoke Test ACO" --cms-id=A9996' | tail -n1)
echo "- ACO ID: $ACO_ID"
CREDS=($(docker exec -it bcda-app_api_1 sh -c 'tmp/bcda generate-client-credentials --cms-id=A9996' | tail -n2))
CLIENT_ID=${CREDS[0]}
CLIENT_SECRET=${CREDS[1]}
echo "- Client ID: $CLIENT_ID"
echo "- Client secret: $CLIENT_SECRET"
go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=ExplanationOfBenefit
go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient
go run bcda_client.go -host=localhost:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Coverage
docker exec -it bcda-app_db_1 sh -c 'psql "postgres://postgres:toor@db:5432/bcda?sslmode=disable" -c "DELETE FROM acos WHERE cms_id = '"'"'A9996'"'"'"'