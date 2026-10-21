locals {
  app          = "bcda"
  service      = "bene-prefs"
  default_tags = module.platform.default_tags
  env          = terraform.workspace

  account_id          = module.platform.aws_caller_identity.account_id
  kms_key_arn_primary = module.platform.kms_alias_primary.target_key_arn
  full_name           = "${local.app}-${local.env}-${local.service}"
  db_sg_name          = "${local.app}-${local.env}-db"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
  ssm_root_map = {
    bene-prefs = "/${local.app}/${local.env}/${local.service}/"
  }
}

data "aws_rds_cluster" "this" {
  cluster_identifier = "${local.app}-${local.env}-aurora"
}

// the role generated for this function is depended on by BFD
// any updates to the role must be coordinated with BFD before deployment
module "bene_prefs_function" {
  source = "github.com/CMSgov/cdap//terraform/modules/function?ref=8a6527c0689bb46ae0e74bd47e4087ab59cff1b0"

  architecture = "arm64"

  name        = local.service
  description = "Ingests the most recent beneficiary bene-prefs list from BFD"

  handler = "bootstrap"
  runtime = "provided.al2023"

  memory_size = 128

  platform = module.platform

  liveness_check_enabled = false

  github_actions_repos = ["bcda-app:*"]

  environment_variables = {
    APP_NAME = local.full_name
    DB_HOST  = "postgres://${data.aws_rds_cluster.this.endpoint}:${data.aws_rds_cluster.this.port}/bcda"
    ENV      = local.env
  }

  function_role_inline_policies = {
    sqs-bene-prefs-bucket-events = data.aws_iam_policy_document.bucket_sqs.json,
    assume-bucket                = data.aws_iam_policy_document.assume_bucket_role.json,
    admin-subscribe-bfd-topic    = data.aws_iam_policy_document.subscribe_bfd_topic.json
  }

  ssm_parameter_paths = [
    "/${local.app}/${local.env}/sensitive/api/DATABASE_URL",
    "/${local.app}/${local.env}/bene-prefs/sensitive/iam_bucket_role_arn"
  ]
}

resource "aws_sqs_queue" "this" {
  content_based_deduplication       = false
  delay_seconds                     = 0
  fifo_queue                        = false
  kms_data_key_reuse_period_seconds = 300
  kms_master_key_id                 = local.kms_key_arn_primary
  name                              = local.full_name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sqs:SendMessage"
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = module.platform.ssm.bene-prefs.sns_topic_arn.value
          }
        }
        Effect = "Allow"
        Principal = {
          Service = "sns.amazonaws.com"
        }
        Resource = "arn:aws:sqs:us-east-1:${local.account_id}:${local.full_name}"
        Sid      = "SnsSendMessage"
      },
      {
        Action = "sns:Subscribe"
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = module.platform.ssm.bene-prefs.sns_topic_arn.value
          }
        }
        Effect = "Allow"
        Principal = {
          Service = "sns.amazonaws.com"
        }
        Resource = "arn:aws:sqs:us-east-1:${local.account_id}:${local.full_name}"
        Sid      = "SnsSubscribe"
      },
    ]
  })
  receive_wait_time_seconds  = 0
  visibility_timeout_seconds = 900
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = aws_sqs_queue.this.arn
  protocol  = "sqs"
  topic_arn = module.platform.ssm.bene-prefs.sns_topic_arn.value
}

resource "aws_lambda_event_source_mapping" "this" {
  event_source_arn = aws_sqs_queue.this.arn
  function_name    = module.bene_prefs_function.name
  batch_size       = 1
  enabled          = true
}
