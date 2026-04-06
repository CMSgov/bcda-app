locals {
  service      = "bene-prefs"
  default_tags = module.platform.default_tags
  env          = terraform.workspace

  account_id            = module.platform.aws_caller_identity.account_id
  kms_key_arn_primary   = module.platform.kms_alias_primary.target_key_arn
  kms_key_arn_secondary = module.platform.kms_alias_secondary.target_key_arn
  name_prefix           = "${local.service_prefix}-${local.service}"
  private_subnets       = nonsensitive(toset(keys(module.platform.private_subnets)))
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

data "aws_security_groups" "db" {
  tags = {
    Name = "bcda-${local.env}-db"
  }
}

resource "aws_security_group_rule" "db" {
  type                     = "ingress"
  from_port                = data.aws_rds_cluster.this.port
  to_port                  = data.aws_rds_cluster.this.port
  protocol                 = "tcp"
  security_group_id        = one([data.aws_security_groups.db.ids])[0]
  source_security_group_id = aws_security_group.this.id
}

# ---------------------------------------------------------------------------
# Managed policies
# ---------------------------------------------------------------------------

data "aws_iam_policy_document" "assume_bucket_role" {
  statement {
    sid       = "AssumeBucketRole"
    actions   = ["sts:AssumeRole"]
    resources = [module.platform.ssm.bene_prefs.iam_bucket_role_arn.value]
  }
}

resource "aws_iam_policy" "assume_bucket_role" {
  name        = "bcda-${local.env}-${local.service}-assume-bucket-role"
  path        = module.platform.iam_defaults.path
  description = "Allows ${local.service} to assume the S3 bucket role from SSM."

  policy = data.aws_iam_policy_document.assume_bucket_role.json
}

data "aws_iam_policy_document" "default_function" {
  statement {
    sid = "SsmSqsLogsEc2"
    actions = [
      "ssm:GetParameters",
      "ssm:GetParameter",
      "sqs:ReceiveMessage",
      "sqs:GetQueueAttributes",
      "sqs:DeleteMessage",
      "logs:PutLogEvents",
      "logs:CreateLogStream",
      "logs:CreateLogGroup",
    ]
    resources = ["*"] #TODO: Consider splitting into discrete statements/policy allowances 
  }
  statement {

    sid = "KmsEncryptDecrypt"
    actions = [
      "kms:GenerateDataKey",
      "kms:Encrypt",
      "kms:Decrypt",
    ]
    resources = [
      local.kms_key_arn_primary,
      local.kms_key_arn_secondary,
    ]
  }
}

resource "aws_iam_policy" "default_function" {
  name        = "bcda-${local.env}-${local.service}-default-function"
  path        = module.platform.iam_defaults.path
  description = "SSM, SQS, CloudWatch Logs, EC2 networking, and KMS permissions for ${local.service}."

  policy = data.aws_iam_policy_document.default_function.json
}

# ---------------------------------------------------------------------------
# IAM role
# ---------------------------------------------------------------------------

resource "aws_iam_role" "this" {
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
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
            module.platform.kion_roles["ct-ado-bcda-application-admin"].arn,
          ]
        }
      },
    ]
  })

  force_detach_policies = true
  name                 = "bcda-${local.env}-${local.service}"
  path                 = module.platform.iam_defaults.path
  permissions_boundary = module.platform.iam_defaults.boundary
}

resource "aws_iam_role_policy_attachment" "this" {
  #TODO: Complexity below is for eventual targeting of `test` and `prod` environments
  for_each = { for k, v in {
    assume_bucket_role = try(aws_iam_policy.assume_bucket_role.arn, "")
    default_function   = try(aws_iam_policy.default_function.arn, "")
  } : k => v if length(v) > 0 }

  role = aws_iam_role.this.name
  policy_arn = each.value
}

module "bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = local.env
  name          = "${local.app}-${local.env}-${local.service}-lambda"
  ssm_parameter = "/${local.app}/${local.env}/${local.service}/nonsensitive/bucket_name"
}

resource "aws_lambda_function" "this" {
  s3_key       = "function-3540b70393e3dc30f375eee2e8635a65c6f21036.zip"
  s3_bucket    = module.bucket.id
  package_type = "Zip"
  handler      = "bootstrap"

  function_name                  = local.name_prefix
  description                    = "Ingests the most recent beneficiary opt-out list from BFD"
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
    ignore_changes = [
      s3_object_version,
      s3_key,
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
  policy = jsonencode({
    Version = "2012-10-17"
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
        Resource = "arn:aws:sqs:us-east-1:${local.account_id}:${local.name_prefix}"
        Sid      = "SnsSendMessage"
      },
    ]
  })
  receive_wait_time_seconds  = 0
  visibility_timeout_seconds = 900
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = aws_sqs_queue.this.arn
  protocol  = "sqs"
  topic_arn = module.platform.ssm.bene_prefs.sns_topic_arn.value
}

resource "aws_lambda_event_source_mapping" "this" {
  event_source_arn = aws_sqs_queue.this.arn
  function_name    = aws_lambda_function.this.function_name
  batch_size       = 1
  enabled          = true
}
