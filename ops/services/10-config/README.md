# BCDA Config Root Module

This root module is responsible for configuring the sops-enabled strategy for storing sensitive and nonsensitive configuration in AWS SSM Parameter Store.
The environment-specific configuration values are located in the `values` directory.

## Usage

### Initial Setup

First, initialize and apply the configuration with the `sopsw` script targeted:

```bash
cd ops/services/10-config
export TF_VAR_env=dev
tofu init
tofu apply -target module.sops.local_file.sopsw[0]
```

### Editing Encrypted Configuration

The `sopsw` script should be automatically generated in the `bin/` directory in the initial setup. You can then edit the encrypted configuration files for each environment:

```bash
# Edit dev environment
./bin/sopsw -e values/dev.sopsw.yaml

# Edit test environment
./bin/sopsw -e values/test.sopsw.yaml

# Edit sandbox environment
./bin/sopsw -e values/sandbox.sopsw.yaml

# Edit prod environment
./bin/sopsw -e values/prod.sopsw.yaml
```

### Deploying Configuration Changes

After editing configuration files, deploy the changes to AWS Parameter Store:

```bash
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
- **tofu** - For deploying configuration to AWS Parameter Store
