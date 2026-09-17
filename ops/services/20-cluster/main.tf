locals {
  app                                   = "bcda"
  default_tags                          = module.platform.default_tags
  env                                   = terraform.workspace
  cdap_env                              = local.parent_env == "prod" || local.env == "sandbox" ? "prod" : "test"
  is_prod                               = contains(["prod", "sandbox"], local.parent_env)
  service                               = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")
  local_zone_name                       = "bcda-${local.env}.local"
  cloudwatch_alarms_topic_name          = "bcda-${local.env}-cloudwatch-alarms"
  cloudwatch_critical_alarms_topic_name = "bcda-${local.env}-cloudwatch-critical-alarms"
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/20-cluster"
  service     = local.service
}

module "ecs_cluster" {
  source                = "github.com/CMSgov/cdap/terraform/modules/cluster?ref=86e705b7a0d81ee1f481948678092ed47ba32741"
  cluster_name_override = "bcda-${local.env}"
  platform              = module.platform
}

resource "aws_route53_zone" "local_zone" {
  name = local.local_zone_name

  vpc {
    vpc_id = module.platform.vpc_id
  }
}

resource "aws_kms_key" "app_config_kms_key" {
  description             = "bcda-${module.platform.env}-app-config-kms"
  deletion_window_in_days = 10
  enable_key_rotation     = true
  policy                  = <<EOF
  {
      "Version": "2012-10-17",
      "Id": "key-default-1",
      "Statement": [
            {
              "Sid": "Enable IAM User Permissions",
              "Effect": "Allow",
              "Principal": {
                  "AWS": "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
              },
              "Action": "kms:*",
              "Resource": "*"
          },
          {
              "Sid": "Enable IAM User Permissions",
              "Effect": "Allow",
              "Principal": {
                  "AWS": "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"
              },
              "Action": ["kms:GenerateDataKey*","kms:Decrypt"],
              "Resource": "*"
          }
      ]
  }
  EOF
}

resource "aws_kms_alias" "app_config_kms_alias" {
  name          = "alias/bcda-${module.platform.env}-app-config-kms"
  target_key_id = aws_kms_key.app_config_kms_key.key_id
}
