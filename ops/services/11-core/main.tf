locals {
  default_tags = module.platform.default_tags
  service      = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = terraform.workspace
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/${basename(abspath(path.module))}"
  service     = local.service
}

##################
# ACCESS LOG KMS #
##################

resource "aws_kms_key" "access_log_kms_key" {
  description             = "bcda-${module.platform.env}-access-log-kms"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "access_log_kms_alias" {
  name          = "alias/bcda-${module.platform.env}-access-log-kms"
  target_key_id = aws_kms_key.access_log_kms_key.key_id
}

##################
# APP CONFIG KMS #
##################

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

###########
# EFS KMS #
###########

resource "aws_kms_key" "efs_kms_key" {
  description             = "bcda-${module.platform.env}-efs"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "efs_kms_alias" {
  name          = "alias/bcda-${module.platform.env}-efs"
  target_key_id = aws_kms_key.efs_kms_key.key_id
}

###########
# RDS KMS #
###########

resource "aws_kms_key" "rds_kms_key" {
  description             = "bcda-${module.platform.env}-rds"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "rds_kms_alias" {
  name          = "alias/bcda-${module.platform.env}-rds"
  target_key_id = aws_kms_key.rds_kms_key.key_id
}
