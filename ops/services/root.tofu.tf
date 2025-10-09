# This root tofu.tf is symlink'd to by all per-env Terraservices. Changes to this tofu.tf apply to
# _all_ Terraservices, so be careful!

locals {
  app              = "bcda"
  established_envs = ["dev", "test", "sandbox", "prod"]
  service_prefix   = "${local.app}-${local.env}"

  parent_env = coalesce(
    var.parent_env,
    one([for x in local.established_envs : x if can(regex("${x}$$", terraform.workspace))]),
    "invalid-parent-environment;do-better"
  )

  state_buckets = {
    dev     = "bcda-dev-tfstate-20250409202710600700000001"
    test    = "bcda-test-tfstate-20250409171646342600000001"
    sandbox = "bcda-sandbox-tfstate-20250416201512973800000001"
    prod    = "bcda-prod-tfstate-20250411203841436200000001"
  }
}

variable "region" {
  default  = "us-east-1"
  nullable = false
  type     = string
}

variable "secondary_region" {
  default  = "us-west-2"
  nullable = false
  type     = string
}

variable "parent_env" {
  description = <<-EOF
  The parent environment of the current solution. Will correspond with `terraform.workspace`".
  Necessary on `tofu init` and `tofu workspace select` _only_. In all other situations, parent env
  will be divined from `terraform.workspace`.
  EOF
  type        = string
  nullable    = true
  default     = null
  validation {
    condition     = var.parent_env == null || one([for x in local.established_envs : x if var.parent_env == x && endswith(terraform.workspace, x)]) != null
    error_message = "Invalid parent environment name."
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = local.default_tags
  }
}

provider "aws" {
  alias = "secondary"

  region = var.secondary_region
  default_tags {
    tags = local.default_tags
  }
}

terraform {
  backend "s3" {
    bucket       = local.state_buckets[local.parent_env]
    key          = "ops/services/${local.service}/tofu.tfstate"
    use_lockfile = true
  }
}
