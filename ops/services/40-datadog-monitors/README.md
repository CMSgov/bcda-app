# OpenTofu for Datadog Monitors

This service sets up the infrastructure for the Datadog Monitors for BCDA.

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

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_datadog"></a> [datadog](#requirement\_datadog) | ~>4.4 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_common_datadog_monitors"></a> [common\_datadog\_monitors](#module\_common\_datadog\_monitors) | github.com/CMSgov/cdap//terraform/modules/datadog_monitors | 06837df7747e3258986d2c89fa4bc31aaf92f29a |
| <a name="module_platform"></a> [platform](#module\_platform) | github.com/CMSgov/cdap//terraform/modules/platform | 941672f97adfd8a19ce6533313302c4c74bac7a8 |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_parent_env"></a> [parent\_env](#input\_parent\_env) | The parent environment of the current solution. Will correspond with `terraform.workspace`".<br/>Necessary on `tofu init` and `tofu workspace select` \_only\_. In all other situations, parent env<br/>will be divined from `terraform.workspace`. | `string` | `null` | no |
| <a name="input_region"></a> [region](#input\_region) | n/a | `string` | `"us-east-1"` | no |
| <a name="input_secondary_region"></a> [secondary\_region](#input\_secondary\_region) | n/a | `string` | `"us-west-2"` | no |

## Outputs

No outputs.

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_datadog"></a> [datadog](#requirement\_datadog) | ~>4.4 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_common_datadog_monitors"></a> [common\_datadog\_monitors](#module\_common\_datadog\_monitors) | github.com/CMSgov/cdap//terraform/modules/datadog_monitors | 945fbd644cc8d239bdf3f3a3a7241fb6066a0f55 |
| <a name="module_platform"></a> [platform](#module\_platform) | github.com/CMSgov/cdap//terraform/modules/platform | 941672f97adfd8a19ce6533313302c4c74bac7a8 |

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_parent_env"></a> [parent\_env](#input\_parent\_env) | The parent environment of the current solution. Will correspond with `terraform.workspace`".<br/>Necessary on `tofu init` and `tofu workspace select` \_only\_. In all other situations, parent env<br/>will be divined from `terraform.workspace`. | `string` | `null` | no |
| <a name="input_region"></a> [region](#input\_region) | n/a | `string` | `"us-east-1"` | no |
| <a name="input_secondary_region"></a> [secondary\_region](#input\_secondary\_region) | n/a | `string` | `"us-west-2"` | no |

## Outputs

No outputs.

<!-- END_TF_DOCS -->
