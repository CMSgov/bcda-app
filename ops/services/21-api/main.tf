locals {
  default_tags = module.platform.default_tags
  service      = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")
  app_sg_name  = "bcda-api-${module.platform.env}"
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = terraform.workspace
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/${basename(abspath(path.module))}"
  service     = local.service
}

/* ------ API/SSAS Security Group ------- */
resource "aws_security_group" "app_sg" {
  name        = local.app_sg_name
  description = "bcda api app security group"
  vpc_id      = module.platform.vpc_id

  tags = {
    Name = local.app_sg_name
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      id,
      description,
    ]
  }
}
