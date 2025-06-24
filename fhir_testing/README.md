# BCDA FHIR Testing

BCDA uses the [Inferno Bulk Data Test Kit](https://github.com/inferno-framework/bulk-data-test-kit) to make sure BCDA conforms to the FHIR Implementation Guide.

## Config
The tests that we want to check are specified in [config.json](./config.json).

Tests that are required to pass to maintain the current level of conformance are the "required_tests". Required tests include:
- TLS Tests (`bulk_data_server_tls_version`)
- Group Export tests (`bulk_data_group_export`...)
- Patient Export tests (`bulk_data_patient_export`...)

Tests that are not required because our current BCDA API does not conform and will not be updated until BCDA v3:
- Group Export Cancel test (`bulk_data_group_export_cancel_group`)
- Group Export Cancel test (`bulk_data_patient_export_cancel_group`)

Tests that are not required because they are not applicable:
- System-level exports (`bulk_data_system_export`...) are not supported by BCDA
- FHIR Validation (...`_export_validation_stu2`...) is outside the scope of BCDA

## Run the Tests
There are two ways to easily run the tests: a make command and a github action workflow (FHIR Scan). Each method will build out the docker containers that hosts the inferno server, then spin up a container to run the script that gets a token, kick off the inferno tests, wait for it to finish, and then interpret the results

### Make fhir_testing
Run FHIR Conformance tests against your local deployment using the make command:

```sh
make fhir_testing
```

### FHIR Scan
The FHIR Scan workflow runs the inferno tests on-demand on the sandbox environment.

## Results
The script returns each test that is configured in the config.json, and whether it passed or failed.

```sh
- PASS - bulk_data_v200-bulk_data_export_tests_v200-bulk_data_server_tests_stu2-bulk_data_server_tls_version_stu2
- PASS - bulk_data_v200-bulk_data_export_tests_v200-bulk_data_group_export_v200-bulk_data_group_export_group_stu2-bulk_data_group_export_operation_support
...
```

and then returns a summary of # of tests passed and failed:

```sh
SUMMARY:
 - Tests Passed: 21
 - Tests Failed: 0
 ```