## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~>6 |
| <a name="requirement_datadog"></a> [datadog](#requirement\_datadog) | ~>4.4 |

## Providers

No providers.

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_datadog_dashboard"></a> [datadog\_dashboard](#module\_datadog\_dashboard) | github.com/CMSgov/cdap//terraform/modules/datadog_dashboard | 945fbd644cc8d239bdf3f3a3a7241fb6066a0f55 |
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
