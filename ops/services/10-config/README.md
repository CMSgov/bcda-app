# BCDA Config Root Module

This root module is responsible for configuring the sops-enabled strategy for storing sensitive and nonsensitive configuration in AWS SSM Parameter Store.
The environment-specific configuration values are located in the `values` directory.  You will need to have copied over AWS short term access keys for all of the following.  See cloudtamer to get keys.

## Usage

### Initial Setup

First, initialize and apply the configuration with the `sopsw` script targeted.

```bash
cd ops/services/10-config
export TF_VAR_env=dev
tofu init
tofu apply -target 'module.sops.local_file.sopsw[0]' -var=create_local_sops_wrapper=true
```

### Editing Encrypted Configuration

The `sopsw` script should be automatically generated in the `bin/` directory in the initial setup. You can then edit the encrypted configuration files for each environment:

```bash
# Edit dev environment, for example
./bin/sopsw -e values/dev.sopsw.yaml
```

### Deploying Configuration Changes

After editing configuration files, deploy the changes to AWS Parameter Store:

```bash
# Review changes before applying
tofu plan -var env=dev

# Apply changes
tofu apply -var env=dev
```

## Configuration Structure

Configuration files follow this pattern:
- `/bcda/${env}/<service>/<sensitivity>/<parameter>`
- Values with `/nonsensitive/` in the path remain unencrypted
- Values with `/sensitive/` in the path are encrypted

### Example Configuration

```yaml
/bcda/${env}/core/sensitive/database_password: "encrypted-password"
/bcda/${env}/core/nonsensitive/database_name: "bcda_dev"
/bcda/${env}/api/sensitive/jwt_secret: "encrypted-jwt"
/bcda/${env}/api/nonsensitive/api_version: "v1"
```

## Dependencies

### Required Tools
- **awscli** - For AWS authentication and KMS operations
- **sops** - For encryption/decryption (`brew install sops`)
- **yq** - For YAML processing (`brew install yq`)
- **envsubst** - For environment variable substitution (`brew install gettext`)

### External Tools
- **tofu** - For deploying configuration to AWS Parameter Store (`brew install opentofu`)

<!-- BEGIN_TF_DOCS -->
<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Providers

No providers.

<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Requirements

No requirements.

<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_env"></a> [env](#input\_env) | The application environment (dev, test, sandbox, prod) | `string` | n/a | yes |
| <a name="input_create_local_sops_wrapper"></a> [create\_local\_sops\_wrapper](#input\_create\_local\_sops\_wrapper) | When `true`, creates sops wrapper file at `bin/sopsw`. | `bool` | `false` | no |
| <a name="input_region"></a> [region](#input\_region) | n/a | `string` | `"us-east-1"` | no |
| <a name="input_secondary_region"></a> [secondary\_region](#input\_secondary\_region) | n/a | `string` | `"us-west-2"` | no |

<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_platform"></a> [platform](#module\_platform) | github.com/CMSgov/cdap//terraform/modules/platform | ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66 |
| <a name="module_sops"></a> [sops](#module\_sops) | github.com/CMSgov/cdap//terraform/modules/sops | 8874310 |

<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Resources

No resources.

<!--WARNING: GENERATED CONTENT with terraform-docs, e.g.
     'terraform-docs --config "$(git rev-parse --show-toplevel)/.terraform-docs.yml" .'
     Manually updating sections between TF_DOCS tags may be overwritten.
     See https://terraform-docs.io/user-guide/configuration/ for more information.
-->
## Outputs

No outputs.
<!-- END_TF_DOCS -->
