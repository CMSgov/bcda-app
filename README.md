# Beneficiary Claims Data API (BCDA)

BCDA gives eligible model entities access to Medicare claims data via FHIR resources.

## About the Project

The Beneficiary Claims Data API (BCDA) follows the Bulk FHIR Implementation Guide to provide eligible model entities access to their attributed enrollees’ Medicare Parts A, B, and D claims data, including data from any care received outside their organization.

## Core Team

A list of core team members responsible for the code and documentation in this repository can be found in [COMMUNITY.md](COMMUNITY.md).

## Documentation

Find API documentation on the [BCDA website](https://bcda.cms.gov/sandbox/user-guide/)

# Development and Software Delivery Lifecycle

## Local Development

The steps below are necessary to run the project.

### Dependencies

To get started, install some dependencies:

1. Install [Go](https://golang.org/doc/install)
2. Install [Docker](https://docs.docker.com/install/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)
4. Install [Ansible Vault](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html) and its dependencies
   For further ansible documentation see: (https://docs.ansible.com/ansible/2.4/vault.html)
5. Install [Pre-commit](https://pre-commit.com/) with [Gitleaks](https://github.com/gitleaks/gitleaks)
6. Ensure all dependencies installed above are on PATH and can be executed directly from command line.

### Secrets

The files committed in the `shared_files/encrypted` directory hold secret information, and are encrypted with [Ansible Vault](https://docs.ansible.com/ansible/2.4/vault.html).

#### Ansible Vault

- Create a file named `.vault_password` in the root directory of the repository.
- You should have been given access to [Box](https://app.box.com).  In Box search for `vault_password.txt`, copy the text and paste it into your `.vault_password` file.

#### Decrypt/Encrypt Secrets

You can now decrypt the encrypted secret files (found at `shared_files/encrypted`) by running the following command from the repository root directory:

```
./ops/secrets --decrypt
```

Copy the decrypted files from `shared_files/encrypted` to the sibling directory `shared_files/decrypted`. After make sure to re-encrypt each secret file in the encrypted folder (as these files are committed):

```
./ops/secrets --encrypt <filename>
```

#### Using Secrets

>**Never put passwords, keys, or secrets of any kind in application code. Instead, use the strategy outlined here:**

1. In the project root `bcda-app/` directory, create a file called `.env.sh`. This file is ignored by git and will not be committed:

```
$ touch .env.sh
```

2. Edit `.env.sh` to include the bash shebang and any necessary environment variables (see `shared_files/decrypted/local.env` as well as section titled 'Environment variables').  It should look like this after:

```
#!/bin/bash
export BCDA_SSAS_CLIENT_ID="<clientID>"
export BCDA_SSAS_SECRET="<clientSecret>"
<other needed env vars>
```

3. Source the file to add the variables to your local development environment:

```
$ source .env.sh
```

Optionally, you can edit your `~/.zshrc` or `~/.bashrc` file to eliminate the need to source the file for each shell start by appending this line:

```
source [src-path]/bcda-app/.env.sh
```

`[src-path]` is your relative path to the bcda-app repo.

### Start the API

#### 1. Build Images

Before we can run the application locally, we need to build the docker images and load the fixtures:

```sh
make docker-bootstrap
```

After that has completed successfully, we can start the containers. Include the `--watch` flag to automatically rebuild the API and worker containers on code changes.

```sh
docker compose up --watch
```

#### 2. Get a token

Once the containers are running, you will need to generate a set of credentials for an ACO so that you can get a token. The loaded fixtures will include some ACOs that have beneficiaries attributed to them already. The ACOs loaded in the previous step are A9994 and A9996, but you can also look in the application database to view and modify more.

```sh
ACO_CMS_ID=<> make credentials
```

This will generate a client ID and secret that can be used to acquire a token from the [System-to-System Authentication Service (SSAS)](https://github.com/CMSgov/bcda-ssas-app):

```sh
curl --location --request POST 'http://localhost:3003/token' \
--header 'Accept: application/json' \
--user '<clientid:secret>'
```

#### 3. Make a request

After we successfully retrieve a token, we can make a request to any of the available endpoints. The PostMan collections under `test/postman_test/...` can be imported into postman directly and used to make requests, or you can use any tool like `curl`.


### Run Tests

**Prerequisite**: Before running the tests and producing test metrics, you must complete the `Build Images` step from `Start the API` section.

#### 1. Seed The Database

>**Note:** `make unit-test` will automatically run the command below, so this step is not necessary if you'd like to just simply run the unit tests.

Spin up the Postgres container & run migrations:

```sh
$ make unit-test-db
```

If you are running any tests that require localstack, spin up localstack as well:

```sh
$ make unit-test-localstack
```

#### 2. Source Environment Variables

Source the required environment variables from the `./.vscode/settings.json` (under go.testEnvVars) and `./shared_files/decrypted/local.env`.

>**Note:** Since we're connecting to Postgres externally, we need to use the local host/port instead.

For vscode users, these variables are already by the workspace settings file (`./.vscode/settings.json`)

>**Note:** If this is the first time running the tests follow instructions in the 'Running unit tests locally' section of this README. Then run:

```sh
make load-fixtures
```

In order to keep the test feedback loop optimized, the following items must be handled by the caller (and are not handled by the test targets):

- Ensuring the compose stack is up and running
- Ensuring the database has been seeded
- Managing images/containers (if Dockerfile changes have occurred, an image rebuild is required and won't occur as part of the test targets)

#### Test Containers

1. Run golang linter and gosec:

```sh
make lint
```

2. Run unit tests (this places results and a coverage report in test_results/<timestamp>):

```sh
make unit-test
```

3. Run postman integration tests:

```sh
make postman env=local maintenanceMode=""
```

4. Run smoke tests:

```sh
make smoke-test env=local maintenanceMode=""
```

5. Run full test suite (executes all of items in 1-4 above):

```sh
make test
```

6. Run performance tests (primarily to be utilized by Jenkins in AWS):

```sh
make performance-test
```

#### Update Database Seed Data

After the user has finished updating the Postgres database used for unit testing with the new data, the user can update
the seed data by running the following command:

```sh
make unit-test-db-snapshot
```

This script will update `./db/testing/docker-entrypoint-initdb.d/dump.pgdata` file.
This file is used to initialize the Postgres db with all of the necessary data needed for the various unit tests.

For more information on intialization, please see `db/testing/docker-entrypoint-initdb.d/01-restore.sh`.
This script is executed when the Postgres container is launched.

>**Note**: The updated `dump.pgdata` should be committed with the other associated changes.

#### Running Single / Single-file Unit Tests

This step assumes that the user has installed VSCode, the Go language extension available [here](https://marketplace.visualstudio.com/items?itemName=golang.Go), and has successfully imported test data to their local database.

To run tests from within VSCode:
In a FILENAME_test.go file, there will be a green arrow to the left of the method name, and clicking this arrow will run a single test locally. Tests should not be dependent upon other tests, but if a known-good test is failing, the user can run all tests in a given file by going to View -> Command Palette -> Go: Test Package, which will run all tests in a given file. Alternatively, in some instances, the init() method can be commented out to enable testing of single functions.

#### Auto-generating mock implementations

Testify mocks can be automatically be generated using [mockery](https://github.com/vektra/mockery). Installation and other runtime instructions can be found [here](https://github.com/vektra/mockery/blob/master/README.md). Mockery uses the `/.mockery.yml` file to define configuration including which interfaces to build.  To regenerate mocks simply run `mockery`.

#### FHIR Conformance Testing

Run FHIR Conformance tests against your local deployment:

```sh
make fhir_testing
```

See FHIR Testing [here](fhir_testing/README.md) for more info on the inferno tests and the FHIR Scan workflow.

### Environment variables

Configure the `bcda` and `bcdaworker` apps by setting the following environment variables.

#### bcda

```
BCDA_ERROR_LOG <file_path>
BCDA_REQUEST_LOG <file_path>
BCDA_BB_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
FHIR_PAYLOAD_DIR <directory_path>
JWT_EXPIRATION_DELTA <integer> (time in hours that JWT access tokens are valid for)
```

#### bcdaworker

```
BCDA_WORKER_ERROR_LOG <file_path>
BCDA_BB_LOG <file_path>
BB_CLIENT_CERT_FILE <file_path>
BB_CLIENT_KEY_FILE <file_path>
BB_SERVER_LOCATION <url>
FHIR_PAYLOAD_DIR <directory_path>
BB_TIMEOUT_MS <integer>
```

### Container Interaction

You can use docker to run commands against the running containers.

**Example:** Use docker to look at the api database with psql.

```sh
docker run --rm --network bcda-app-net -it postgres psql -h bcda-app-db-1 -U postgres bcda
```

**Example:** See docker-compose.yml for the password.

Use docker to run the CLI against an API instance

```
docker exec -it bcda-app-api-1 sh -c 'bcda -h'
```

### IDE Setup

#### vscode

Follow installing go + vscode [setup guide](https://marketplace.visualstudio.com/items?itemName=golang.go#getting-started).
Additional settings found under `.vscode/settings.json` allow tests to be run within vscode.

#### Debugging

To attach to either the api or worker process within its respective docker container and debug, stop at breakpoints, and view local variables during execution, run the appropriate debug command.

`make debug-api`

or

`make debug-worker`

Once the containers are up, the program to debug will wait to be connected to dlv. Use dlv directly, or use a remote-attach debugging configuration in vscode.
```
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to server",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "port": 4040,
            "host": "127.0.0.1"
        }
    ]
}
```
## Coding Style and Linters

<!-- TODO - Add the repo's linting and code style guidelines -->

## Branching Model

<!--- TODO - Add the repo's branching model -->

## Contributing

Thank you for considering contributing to an Open Source project of the US Government! For more information about our contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

### Pre-commit

Anyone committing to this repo must use the pre-commit hook to lower the likelihood that secrets will be exposed. You can install pre-commit using the MacOS package manager `Homebrew` below, or use installation options that can be found in the [pre-commit documentation](https://pre-commit.com/#install):

```sh
brew install pre-commit
```

Before you can install the `hooks`, you will need to manually install `goimports`::

```
go install golang.org/x/tools/cmd/goimports@latest
```

After that is installed, we can, install the hooks:

```sh
pre-commit install
```

This will download and install the pre-commit hooks specified in `.pre-commit-config.yaml`, which includes gitleaks for secret scanning and go-imports to ensure that any added, copied, or modified go files are formatted properly.

### Go Modules

The project uses [Go Modules](https://golang.org/ref/mod) allowing you to clone the repo outside of the `$GOPATH`. This also means that running `go get` inside the repo will add the dependency to the project, not globally.

## Community

The BCDA team is taking a community-first and open source approach to the product development of this tool. We believe government software should be made in the open and be built and licensed such that anyone can download the code, run it themselves without paying money to third parties or using proprietary software, and use it as they will.

We know that we can learn from a wide variety of communities, including those who will use or will be impacted by the tool, who are experts in technology, or who have experience with similar technologies deployed in other spaces. We are dedicated to creating forums for continuous conversation and feedback to help shape the design and development of the tool.

We also recognize capacity building as a key part of involving a diverse open source community. We are doing our best to use accessible language, provide technical and process documents, and offer support to community members with a wide variety of backgrounds and skillsets.

### Community Guidelines

Principles and guidelines for participating in our open source community are can be found in [COMMUNITY.md](COMMUNITY.md). Please read them before joining or starting a conversation in this repo or one of the channels listed below. All community members and participants are expected to adhere to the community guidelines and code of conduct when participating in community spaces including: code repositories, communication channels and venues, and events.

## Feedback

If you have ideas for how we can improve or add to our capacity building efforts and methods for welcoming people into our community, please let us know at **bcapi@cms.hhs.gov**. If you would like to comment on the tool itself, please let us know by filing an **issue on our GitHub repository.**

## Policies

### Open Source Policy

We adhere to the [CMS Open Source
Policy](https://github.com/CMSGov/cms-open-source-policy). If you have any
questions, just [shoot us an email](mailto:opensource@cms.hhs.gov).

### Security and Responsible Disclosure Policy

_Submit a vulnerability:_ Vulnerability reports can be submitted through [Bugcrowd](https://bugcrowd.com/cms-vdp). Reports may be submitted anonymously. If you share contact information, we will acknowledge receipt of your report within 3 business days.

For more information about our Security, Vulnerability, and Responsible Disclosure Policies, see [SECURITY.md](SECURITY.md).

### Software Bill of Materials (SBOM)

A Software Bill of Materials (SBOM) is a formal record containing the details and supply chain relationships of various components used in building software.

In the spirit of [Executive Order 14028 - Improving the Nation’s Cyber Security](https://www.gsa.gov/technology/it-contract-vehicles-and-purchasing-programs/information-technology-category/it-security/executive-order-14028), a SBOM for this repository is provided here: https://github.com/{{ cookiecutter.project_org }}/{{ cookiecutter.project_repo_name }}/network/dependencies.

For more information and resources about SBOMs, visit: https://www.cisa.gov/sbom.

## Public domain

This project is in the public domain within the United States, and copyright and related rights in the work worldwide are waived through the [CC0 1.0 Universal public domain dedication](https://creativecommons.org/publicdomain/zero/1.0/) as indicated in [LICENSE](LICENSE).

All contributions to this project will be released under the CC0 dedication. By submitting a pull request or issue, you are agreeing to comply with this waiver of copyright interest.