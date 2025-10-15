locals {
  default_tags = module.platform.default_tags
  service      = "config"
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
}

module "sops" {
  source = "github.com/CMSgov/cdap//terraform/modules/sops?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  platform = module.platform
}

output "edit" {
  value = module.sops.sopsw
}
