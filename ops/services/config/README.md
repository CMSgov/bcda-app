# BCDA Config Root Module

This root module is responsible for configuring the sops-enabled strategy for storing sensitive and nonsensitive configuration in AWS SSM Parameter Store.
The _parent environment_ specific configuration values are located in the `values` directory.

## Usage

### Prerequisites

Before editing encrypted configuration files, you need to:

1. **Get the sopsw script** from the CDAP repository.

2. **Deploy the Terraform configuration**:
   ```bash
   cd ops/services/config
   tofu init
   tofu apply
   ```

### Editing Encrypted Configuration

To edit the encrypted configuration files for each environment:

```bash
# Edit dev environment
./bin/sopsw -e values/dev.sopsw.yaml

# Edit test environment
./bin/sopsw -e values/test.sopsw.yaml

# Edit prod environment
./bin/sopsw -e values/prod.sopsw.yaml
```

### Deploying Configuration

```bash
# Initialize Terraform
terraform init

# Plan changes
terraform plan

# Apply changes
terraform apply
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
- **sopsw script** - From CDAP
- **Terraform** - For deploying configuration to AWS Parameter Store
