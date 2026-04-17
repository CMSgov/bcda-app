# OpenTofu for api-waf-sync function and associated infra

This service sets up the infrastructure for the api-waf-sync lambda function in dev for BCDA/DPC.

## Manual deploy / Applying Changes
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
## Automated deploy

This OpenTofu is automatically applied on merge to main by the waf-sync-apply.yml workflow.
