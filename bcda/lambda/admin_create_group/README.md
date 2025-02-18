The Create Group administrative task lambda will add an existing ACO to a specified group.  It should be called via AWS's lambda interface (see: https://confluence.cms.gov/display/BCDA/How+To+deny+an+ACO+From+Generating+Credentials).

You can run the unit test suite from the base dir (bcda-app) using the following command:

make test-path TEST_PATH="bcda/lambda/admin_create_group/*.go". (You might have to make load-fixtures first). It also has an integration test run via github actions (see .github/workflows/admin-aco-deny-integration-test.yml).
The lambda is deployed (or promoted in the case of prod) using github actions (see .github/workflows/admin-create-group-lambda-{env}-deploy.yml files).
