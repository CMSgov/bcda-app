locals {
  service      = "eft-nextgen"
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
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/20-eft-nextgen"
  service     = local.service
  ssm_root_map = {
    eft-nextgen = "/bcda/${local.env}/eft-nextgen/"
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

data "aws_iam_role" "this"{
  name = "bcda-${local.env}-cclf-import-function"
}
# ---------------------------------------------------------------------------
# Managed policies
# ---------------------------------------------------------------------------
data "aws_iam_policy_document" "assume_bucket_role" {
  statement {
    sid       = "AssumeBucketRole"
    actions   = ["sts:AssumeRole"]
    resources = [data.aws_iam_role.this.arn]
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
  role = aws_iam_role.this.name
  policy_arn = aws_iam_policy.default_function.arn
}

module "bucket" {
  source = "github.com/CMSgov/cdap//terraform/modules/bucket?ref=787224b"

  app           = local.app
  env           = local.env
  name          = "${local.app}-${local.env}-${local.service}-lambda"
  ssm_parameter = "/${local.app}/${local.env}/${local.service}/nonsensitive/bucket_name"
}

data "aws_lambda_function" "this" {
  function_name                  = "bcda-${local.env}-cclf-import"
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

data "aws_sqs_queue" "this"{
  name = "${local.app}-${local.env}-cclf-import"
}

resource "aws_sns_topic" "eft_nextgen_topic" {
  display_name      = local.service
  name              = local.service
  kms_master_key_id = "alias/${module.platform.app}-${module.platform.env}"
}

resource "aws_sns_topic_subscription" "this" {
  endpoint  = data.aws_sqs_queue.this.arn
  protocol  = "sqs"
  topic_arn = aws_sns_topic.eft_nextgen_topic.arn
}

resource "aws_lambda_event_source_mapping" "this" {
  event_source_arn = data.aws_sqs_queue.this.arn
  function_name    = data.aws_lambda_function.this.function_name
  batch_size       = 1
  enabled          = true
}
