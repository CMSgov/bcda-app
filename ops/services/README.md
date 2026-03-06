# BCDA Terraservices
- each directory herein is an opinionated module for terraform (tofu)
- each module is terraform workspace-enabled, unless otherwise specified
- the two-digit numbers for each module is significant and identifies the sequence of operations for ordered module application

## Requirements
Some of these modules have requirements for local development or remote CI. Those are including (but not limited to):
- jq *and* yq for manipulating json and yaml documents
- awscli

## Applying Changes
Applying changes for these modules requires initialization of the state **and** selection of the appropriate environmental workspace.
Both can be achieved with the following commands:

```sh
### prod environment
TF_WORKSPACE=default tofu init -var parent_env=prod -reconfigure && tofu workspace select -var parent_env=prod -or-create prod

### sandbox environment
TF_WORKSPACE=default tofu init -var parent_env=sandbox -reconfigure && tofu workspace select -var parent_env=sandbox -or-create sandbox

### test environment
TF_WORKSPACE=default tofu init -var parent_env=test -reconfigure && tofu workspace select -var parent_env=test -or-create test

### dev environment
TF_WORKSPACE=default tofu init -var parent_env=dev -reconfigure && tofu workspace select -var parent_env=dev -or-create dev
```

After the state has been initialized and the workspace selected, users or automation can apply changes by running `tofu apply`.
