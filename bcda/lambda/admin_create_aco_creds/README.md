The Create ACO Credentials administrative task lambda will create and return a new client id and secret for the specified ACO.  It should be called via AWS's lambda interface see: [Manually Create Production Credentials](https://confluence.cms.gov/x/ApF9T).

You can run the unit test suite from the base dir (bcda-app) using the following command:

make test-path TEST_PATH="bcda/lambda/admin_create_aco_creds/*.go". (You might have to make load-fixtures first). It also has an integration test run via github actions (see .github/workflows/admin-create-aco-creds-integration-test.yml).
The lambda is deployed (or promoted in the case of prod) using github actions (see .github/workflows/admin-create-aco-creds-lambda-{env}-deploy.yml files).
