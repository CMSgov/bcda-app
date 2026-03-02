locals {
  service = "config"
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
  source = "github.com/CMSgov/cdap//terraform/modules/sops?ref=8874310"

  platform                  = module.platform
  create_local_sops_wrapper = var.create_local_sops_wrapper
}
