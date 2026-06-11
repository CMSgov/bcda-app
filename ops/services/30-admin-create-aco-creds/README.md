# OpenTofu for Admin Create ACO Credentials function and associated infra

This service sets up the infrastructure for the Admin Create ACO Credentials lambda function in upper and lower environments for BCDA.

## Manual deploy
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

## Automated deploy

This terraform is automatically applied on merge to main by the admin-create-aco-creds-apply.yml workflow.
