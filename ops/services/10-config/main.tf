terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5"
    }
  }
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/config"
  service     = local.service
}

locals {
  default_tags = module.platform.default_tags
  env          = terraform.workspace
  service      = "config"
}

module "sops" {
  source = "github.com/CMSgov/cdap//terraform/modules/sops?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  platform = module.platform
}

output "edit" {
  value = module.sops.sopsw
}
