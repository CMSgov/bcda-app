# OpenTofu for api-waf-sync function and associated infra

This service sets up the infrastructure for the api-waf-sync lambda function in dev for BCDA/DPC.

## Manual deploy

Pass in a backend file when running tofu init. See variables.tf for variables to include. Example:

```bash
export TF_VAR_env=dev
tofu init -backend-config=../../backends/dpc-dev.s3.tfbackend
tofu apply
```

## Automated deploy

This OpenTofu is automatically applied on merge to main by the waf-sync-apply.yml workflow.
