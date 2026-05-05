data "aws_region" "current" {}

data "aws_caller_identity" "current" {
}

data "aws_iam_policy_document" "insights_policy" {
  statement {
    actions = [
      "glue:GetTable",
      "glue:GetTableVersion",
      "glue:GetTableVersions"

    ]
    resources = [
      "*"
    ]
  }

  statement {
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObjectAcl",
      "s3:PutObject"

    ]
    resources = [
      "arn:aws:s3:::bfd-insights-bcda-577373831711",
      "arn:aws:s3:::bfd-insights-bcda-577373831711/*"
    ]
  }

  statement {
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:DescribeKey"

    ]
    resources = [
      "arn:aws:kms:${data.aws_region.current.name}:577373831711:key/a4ca59d6-3978-434b-9a5c-c6ae8896db18"
    ]
  }

  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
      "logs:DescribeLogsGroups",
      "logs:DescribeLogsStreams"

    ]
    resources = [
      "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/bfd-insights-bcda-577373831711:log-stream:*"
    ]
  }

  statement {
    actions = [
      "kinesis:DescribeStream",
      "kinesis:GetShardIterator",
      "kinesis:GetRecords",
      "kinesis:ListShards"

    ]
    resources = [
      "arn:aws:kinesis:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:stream/*"
    ]
  }
}

data "aws_iam_policy" "developer_boundary_policy" {
  name = "developer-boundary-policy"
}

resource "aws_iam_role" "insights_role" {
  name                 = "bcda-${var.env}-bfd-insights"
  description          = "IAM role for managing insights-related resources within the delegatedadmin/developer/ path and permissions boundary."
  path                 = "/delegatedadmin/developer/"
  permissions_boundary = data.aws_iam_policy.developer_boundary_policy.arn
  assume_role_policy   = data.aws_iam_policy_document.insights_role.json
}

data "aws_iam_policy_document" "insights_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["firehose.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values = [
        data.aws_caller_identity.current.account_id
      ]
    }
  }
}

resource "aws_iam_policy" "insights_policy" {
  name_prefix = "bcda-${var.env}-bfd-insights-"
  description = "Policy to access BFD resources within the /delegatedadmin/developer/ path."
  path        = "/delegatedadmin/developer/"
  policy      = data.aws_iam_policy_document.insights_policy.json
}

resource "aws_iam_role_policy_attachment" "insights_role_attachment" {
  role       = aws_iam_role.insights_role.name
  policy_arn = aws_iam_policy.insights_policy.arn
}

module "insights_data_sampler" {
  source            = "../modules/insights_data_sampler_lambda"
  env               = var.env
  db_subnet_group   = var.db_subnet_group
  security_group_id = var.worker_security_group # bcda-worker-dev
}

#
# Insights Event Processors for each environment
# (infrastructure required to send events to Insights from application code)
#
module "event_processor" {
  source            = "../modules/insights_data_event_processor"
  name              = "event_processor"
  description       = "processes real-time events (via application code) and sends to BFD Insights"
  env               = var.env
  insights_role_arn = aws_iam_role.insights_role.arn
}
