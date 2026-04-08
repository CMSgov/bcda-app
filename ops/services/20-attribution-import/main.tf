locals {
  service      = "attribution-import"
  default_tags = module.platform.default_tags
  env          = terraform.workspace

  account_id            = module.platform.aws_caller_identity.account_id
  kms_key_arn_primary   = module.platform.kms_alias_primary.target_key_arn
  kms_key_arn_secondary = module.platform.kms_alias_secondary.target_key_arn
  name_prefix           = "${local.service_prefix}-${local.service}"
  private_subnets       = nonsensitive(toset(keys(module.platform.private_subnets)))
  lambda_filename       = "lambda_function.zip"
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = local.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/20-attribution-import"
  service     = local.service
  ssm_root_map = {
    attribution-import = "/bcda/${local.env}/attribution-import/"
  }
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/bcda-${local.env}-${local.service}"
  retention_in_days = 180
  skip_destroy      = true

  tags = {
    Name = "/aws/lambda/bcda-${local.env}-${local.service}"
  }
}

# ---------------------------------------------------------------------------
# File Bucket
# ---------------------------------------------------------------------------

module "attribution-import_file_bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = local.env
  name          = "${local.app}-${local.env}-${local.service}-file"
  ssm_parameter = "/${local.app}/${local.env}/${local.service}/nonsensitive/file_bucket_name"
}

# ---------------------------------------------------------------------------
# Lambda Bucket
# ---------------------------------------------------------------------------

module "attribution-import_lambda_bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = local.env
  name          = "${local.app}-${local.env}-${local.service}-lambda"
  ssm_parameter = "/${local.app}/${local.env}/${local.service}/nonsensitive/lambda_bucket_name"
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
  name   = local.name_prefix
  tags   = { Name = local.name_prefix }
  vpc_id = module.platform.vpc_id
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
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = module.platform.ssm.attribution-import.misp-eft-role_arn.value
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
  name                  = "bcda-${local.env}-${local.service}"
  path                  = module.platform.iam_defaults.path
  permissions_boundary  = module.platform.iam_defaults.boundary
}

resource "aws_iam_role_policy_attachment" "this" {
  role       = aws_iam_role.this.name
  policy_arn = aws_iam_policy.default_function.arn
}

resource "aws_iam_role_policy_attachment" "vpc_access" {
  role       = aws_iam_role.this.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

# ---------------------------------------------------------------------------
# Managed policies
# ---------------------------------------------------------------------------
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
    resources = ["*"]
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

resource "aws_iam_role_policy_attachment" "default_function" {
  role       = aws_iam_role.this.name
  policy_arn = aws_iam_policy.default_function.arn
}


# -------------------------------------------------------
# IAM Policy — S3 Read/Write on the attribution-import file bucket
# -------------------------------------------------------
data "aws_iam_policy_document" "attribution-import_bucket_rw" {

  statement {
    sid    = "ListBucket"
    effect = "Allow"

    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation",
    ]

    resources = [
      module.attribution-import_file_bucket.arn,
    ]
  }

  statement {
    sid    = "ReadWriteObjects"
    effect = "Allow"

    actions = [
      # Read
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:GetObjectTagging",
      "s3:PutObject",
      "s3:PutObjectTagging",
      "s3:DeleteObject",
      "s3:DeleteObjectVersion",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts",
    ]

    resources = [
      "${module.attribution-import_file_bucket.arn}/*",
    ]
  }
}

resource "aws_iam_policy" "attribution-import_file_bucket" {
  name        = "${local.app}-${local.env}-${local.service}-attribution-import-bucket-rw"
  description = "Read/write access to the ${local.app}-${local.env}-${local.service} attribution-import file bucket"
  policy      = data.aws_iam_policy_document.attribution-import_bucket_rw.json
}

resource "aws_iam_role_policy_attachment" "attribution-import_file_bucket_rw" {
  role       = aws_iam_role.this.name
  policy_arn = aws_iam_policy.attribution-import_file_bucket.arn
}



# -------------------------------------------------------
# attribution-import Lambda
# -------------------------------------------------------
resource "aws_s3_object" "dummy_file_upload" {
  bucket = module.attribution-import_lambda_bucket.id
  key    = local.lambda_filename
  source = "${path.module}/${local.lambda_filename}"
}

resource "aws_lambda_function" "this" {
  s3_bucket         = aws_s3_object.dummy_file_upload.bucket
  s3_key            = aws_s3_object.dummy_file_upload.key
  s3_object_version = aws_s3_object.dummy_file_upload.version_id
  package_type = "Zip"
  handler      = "bootstrap"

  function_name                  = local.name_prefix
  description                    = "Ingests the most recent attribution import from BFD"
  kms_key_arn                    = local.kms_key_arn_primary
  memory_size                    = 128
  reserved_concurrent_executions = 1
  role                           = aws_iam_role.this.arn
  runtime                        = "provided.al2023"
  skip_destroy                   = false
  timeout                        = 900
  architectures = [
    "arm64",
  ]

  tags = {
    code = "https://github.com/CMSgov/bcda-app/tree/main/bcda/lambda/attribution-import"
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
  description              = "attribution-import lambda access"
  security_group_id        = one([data.aws_security_groups.db.ids])[0]
  source_security_group_id = aws_security_group.this.id
}

# -------------------------------------------------------
# attribution-import SQS and SNS
# -------------------------------------------------------

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
            "aws:SourceArn" = aws_sns_topic.attribution-import_nextgen_topic.arn
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

resource "aws_sns_topic" "attribution-import_nextgen_topic" {
  display_name      = "bfd-${local.env}-attribution-import-inbound-received-s3-bcda"
  name              = "bfd-${local.env}-attribution-import-inbound-received-s3-bcda"
  kms_master_key_id = "alias/${module.platform.app}-${module.platform.env}"
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = aws_sqs_queue.this.arn
  protocol  = "sqs"
  topic_arn = aws_sns_topic.attribution-import_nextgen_topic.arn
}

resource "aws_lambda_event_source_mapping" "this" {
  event_source_arn = aws_sqs_queue.this.arn
  function_name    = aws_lambda_function.this.function_name
  batch_size       = 1
  enabled          = true
}

data "aws_iam_policy_document" "sqs_s3_policy" {
  statement {
    sid    = "AllowS3ToSendMessage"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }

    actions   = ["sqs:SendMessage"]
    resources = [aws_sqs_queue.this.arn]

    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [module.attribution-import_file_bucket.arn]
    }
  }
}

resource "aws_sqs_queue_policy" "this" {
  queue_url = aws_sqs_queue.this.id
  policy    = data.aws_iam_policy_document.sqs_s3_policy.json
}

resource "aws_s3_bucket_notification" "this" {
  bucket = module.attribution-import_file_bucket.id

  depends_on = [aws_sqs_queue_policy.this]

  queue {
    queue_arn = aws_sqs_queue.this.arn
    events    = ["s3:ObjectCreated:*"]
  }
}
