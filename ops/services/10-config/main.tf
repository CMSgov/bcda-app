module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = "config"
}

module "sops" {
  source = "github.com/CMSgov/cdap//terraform/modules/sops?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  platform = module.platform
}

output "edit" {
  value = module.sops.sopsw
}
