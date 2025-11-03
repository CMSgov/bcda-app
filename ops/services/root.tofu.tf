# This root tofu.tf is symlink'd to by all per-env Terraservices. Changes to this tofu.tf apply to
# _all_ Terraservices, so be careful!

variable "env" {
  description = "The application environment (dev, test, sandbox, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "test", "sandbox", "prod"], var.env)
    error_message = "Valid value for env is dev, test, sandbox, or prod."
  }
}

variable "region" {
  default  = constants.DefaultRegion
  nullable = false
  type     = string
}

variable "secondary_region" {
  default  = "us-west-2"
  nullable = false
  type     = string
}

locals {
  app            = "bcda"
  env            = var.env
  service_prefix = "${local.app}-${local.env}"

  state_buckets = {
    dev     = "bcda-dev-tfstate-20250409202710600700000001"
    test    = "bcda-test-tfstate-20250409171646342600000001"
    sandbox = "bcda-sandbox-tfstate-20250416201512973800000001"
    prod    = "bcda-prod-tfstate-20250411203841436200000001"
  }
}

provider "aws" {
  region = var.region
}

provider "aws" {
  alias  = "secondary"
  region = var.secondary_region
}

terraform {
  backend "s3" {
    bucket       = local.state_buckets[local.env]
    key          = "ops/services/${local.service}/tofu.tfstate"
    use_lockfile = true
    region       = var.region
  }
}
