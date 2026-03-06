locals {
  service = "bene-prefs"

  account_id            = module.platform.aws_caller_identity.account_id
  kms_key_arn_primary   = module.platform.kms_alias_primary.target_key_arn
  kms_key_arn_secondary = module.platform.kms_alias_secondary.target_key_arn
  name_prefix           = "${local.app}-${local.env}-${local.service}"
  private_subnets       = nonsensitive(toset(keys(module.platform.private_subnets)))
  vpc_id                = module.platform.vpc_id
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/10-config"
  service     = local.service
  ssm_root_map = {
    bene_prefs = "/bcda/${local.env}/bene_prefs/"
  }
}

data "aws_rds_cluster" "this" {
  cluster_identifier = "${local.app}-${local.env}-aurora"
}

resource "aws_iam_role" "this" {
  assume_role_policy = jsonencode(
    {
      Statement = [
        {
          Action = "sts:AssumeRole"
          Effect = "Allow"
          Principal = {
            Service = "lambda.amazonaws.com"
          }
        },
        {
          Action = [
            "sts:TagSession",
            "sts:AssumeRoleWithWebIdentity",
          ]
          Condition = {
            StringEquals = {
              "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
            }
            StringLike = {
              "token.actions.githubusercontent.com:sub" = "repo:CMSgov/bcda-app:*"
            }
          }
          Effect = "Allow"
          Principal = {
            Federated = "arn:aws:iam::${local.account_id}:oidc-provider/token.actions.githubusercontent.com"
          }
        },
        {
          Action = [
            "sts:TagSession",
            "sts:AssumeRole",
          ]
          Effect = "Allow"
          Principal = {
            AWS = [
              module.platform.kion_roles["ct-ado-dasg-application-admin"].arn,
              module.platform.kion_roles["ct-ado-bcda-application-admin"].arn
            ]
          }
        },
      ]
      Version = "2012-10-17"
    }
  )
  force_detach_policies = true
  managed_policy_arns   = [] #FIXME: populate with standalone policies that were once in-line
  name                  = "bcda-${local.env}-${local.service}"
  path                  = module.platform.iam_defaults.path
  permissions_boundary  = module.platform.iam_defaults.boundary

  #FIXME: convert into policy
  inline_policy {
    name = "assume-bucket-role"
    policy = jsonencode(
      {
        Statement = [
          {
            Action   = "sts:AssumeRole"
            Effect   = "Allow"
            Resource = module.platform.ssm.bene_prefs.iam_bucket_role_arn.value
          },
        ]
        Version = "2012-10-17"
      }
    )
  }
  #FIXME: convert into appropriately scoped policy
  inline_policy {
    name = "default-function"
    policy = jsonencode(
      {
        Statement = [
          {
            Action = [
              "ssm:GetParameters",
              "ssm:GetParameter",
              "sqs:ReceiveMessage",
              "sqs:GetQueueAttributes",
              "sqs:DeleteMessage",
              "logs:PutLogEvents",
              "logs:CreateLogStream",
              "logs:CreateLogGroup",
              "ec2:DescribeNetworkInterfaces",
              "ec2:DescribeAccountAttributes",
              "ec2:DeleteNetworkInterface",
              "ec2:CreateNetworkInterface",
            ]
            Effect   = "Allow"
            Resource = "*"
          },
          {
            Action = [
              "kms:GenerateDataKey",
              "kms:Encrypt",
              "kms:Decrypt",
            ]
            Effect = "Allow"
            Resource = [
              local.kms_key_arn_primary,
              local.kms_key_arn_secondary
            ]
          },
        ]
        Version = "2012-10-17"
      }
    )
  }
}

module "bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = local.env
  name          = "${local.app}-${local.env}-${local.service}"
  ssm_parameter = "/${local.app}/${local.env}/${local.service}/nonsensitive/bucket_name"
}

resource "aws_lambda_function" "this" {
  s3_key       = "function-3540b70393e3dc30f375eee2e8635a65c6f21036.zip"
  s3_bucket    = module.bucket.id
  package_type = "Zip"
  handler      = "bootstrap"

  function_name                  = local.name_prefix
  description                    = "Ingests the most recent beneficiary opt-out list from BFD" #FIXME
  kms_key_arn                    = local.kms_key_arn_primary
  memory_size                    = 128
  reserved_concurrent_executions = 1
  role                           = aws_iam_role.this.arn
  runtime                        = "provided.al2023"
  skip_destroy                   = false
  timeout                        = 900
  architectures = [
    "x86_64",
  ]

  tags = {
    code = "https://github.com/CMSgov/bcda-app/tree/main/bcda/lambda/optout"
  }

  lifecycle {
    # FIXME: As of this writing, delivery of the opt-out function is separate from deployment of this module.
    # As such, we must ignore the specific s3_key and s3_object_version configuration.
    ignore_changes = [
      s3_object_version,
      s3_key
    ]
  }

  environment {
    variables = {
      APP_NAME = local.name_prefix
      DB_HOST  = "postgres://${data.aws_rds_cluster.this.endpoint}:${data.aws_rds_cluster.this.port}/bcda"
      ENV      = local.env
    }
  }

  ephemeral_storage {
    size = 512
  }

  logging_config {
    log_format = "Text"
    log_group  = "/aws/lambda/bcda-${local.env}-${local.service}"
  }

  tracing_config {
    mode = "Active"
  }

  vpc_config {
    ipv6_allowed_for_dual_stack = false
    security_group_ids          = [aws_security_group.this.id]
    subnet_ids                  = local.private_subnets
  }
}

resource "aws_security_group" "this" {
  description = "Temporary SG for ${local.name_prefix}"
  egress = [
    {
      cidr_blocks = [
        "0.0.0.0/0",
      ]
      description = ""
      from_port   = 0
      ipv6_cidr_blocks = [
        "::/0",
      ]
      prefix_list_ids = []
      protocol        = "-1"
      security_groups = []
      self            = false
      to_port         = 0
    },
  ]
  name = local.name_prefix
  tags = { Name = local.name_prefix }
}

resource "aws_sqs_queue" "this" {
  content_based_deduplication       = false
  delay_seconds                     = 0
  fifo_queue                        = false
  kms_data_key_reuse_period_seconds = 300
  kms_master_key_id                 = local.kms_key_arn_primary
  name                              = local.name_prefix
  policy = jsonencode(
    {
      Statement = [
        {
          Action = "sqs:SendMessage"
          Condition = {
            ArnEquals = {
              "aws:SourceArn" = module.platform.ssm.bene_prefs.sns_topic_arn.value
            }
          }
          Effect = "Allow"
          Principal = {
            Service = "sns.amazonaws.com"
          }
          Resource = "arn:aws:sqs:us-east-1:${local.account_id}:${local.name_prefix}" #TODO
          Sid      = "SnsSendMessage"
        },
      ]
      Version = "2012-10-17"
    }
  )
  receive_wait_time_seconds  = 0
  visibility_timeout_seconds = 900
}
}
