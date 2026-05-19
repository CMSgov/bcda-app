locals {
  zip_file = "insights_data_sampler.zip"
}

data "aws_ssm_parameter" "slack_webhook_url" {
  name = "/bcda/lambda/slack_webhook_url"
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

resource "aws_iam_role" "insights_data_sampler_role" {
  name_prefix        = "bcda-${var.env}-bfd-insights-sampler-"
  assume_role_policy = data.aws_iam_policy_document.insights_data_sampler_role_policy.json
}

data "aws_iam_policy_document" "insights_data_sampler_role_policy" {
  statement {
    effect = "Allow"
    actions = [
      "sts:AssumeRole",
    ]

    principals {
      type = "Service"
      identifiers = [
        "lambda.amazonaws.com"
      ]
    }
  }
}

resource "aws_iam_role_policy_attachment" "aws_lambda_basic_exec_policy" {
  role       = aws_iam_role.insights_data_sampler_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

data "aws_db_subnet_group" "db_subnet_group" {
  name = var.db_subnet_group
}

resource "aws_kms_key" "cloudwatch_kms_key" {
  description             = "bcda-insights-data-sampler-${var.env}-key"
  deletion_window_in_days = 10
  enable_key_rotation     = true

  policy = <<EOF
{
   "Version":"2012-10-17",
   "Id":"bcda-insights-data-sampler-${var.env}-key-policy",
   "Statement":[
      {
         "Sid":"default",
         "Effect":"Allow",
         "Principal":{
            "AWS":"arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
         },
         "Action":"kms:*",
         "Resource":"*"
      },
      {
         "Effect":"Allow",
         "Principal":{
            "Service":"logs.${data.aws_region.current.name}.amazonaws.com"
         },
         "Action":[
            "kms:Encrypt*",
            "kms:Decrypt*",
            "kms:ReEncrypt*",
            "kms:GenerateDataKey*",
            "kms:Describe*"
         ],
         "Resource":"*",
         "Condition": {
           "ArnEquals": {
              "kms:EncryptionContext:aws:logs:arn": "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/${aws_lambda_function.insights_data_sampler.function_name}"
           }
         }
      }
   ]
}
EOF

}

resource "aws_kms_alias" "cloudwatch_kms_key_alias" {
  name          = "alias/bcda-insights-data-sampler-${var.env}-key"
  target_key_id = aws_kms_key.cloudwatch_kms_key.key_id
}

resource "aws_cloudwatch_log_group" "insights_data_sampler" {
  name              = "/aws/lambda/${aws_lambda_function.insights_data_sampler.function_name}"
  retention_in_days = 14
  kms_key_id        = aws_kms_key.cloudwatch_kms_key.arn
}

resource "aws_lambda_function" "insights_data_sampler" {
  #ts:skip=AWS.LambdaFunction.EncryptionandKeyManagement.0471 - These environment variables are set via AWS console after it is initialized with a blank string from Terraform
  filename         = "${path.module}/${local.zip_file}"
  description      = "Samples BCDA ${var.env} database and puts data in a firehose"
  function_name    = "insights_data_sampler_${var.env}"
  role             = aws_iam_role.insights_data_sampler_role.arn
  handler          = "index.handler"
  source_code_hash = filebase64sha256("${path.module}/${local.zip_file}")
  runtime          = "nodejs22.x"
  timeout          = 90

  vpc_config {
    subnet_ids         = data.aws_db_subnet_group.db_subnet_group.subnet_ids
    security_group_ids = [var.security_group_id]
  }

  environment {
    variables = {
      SLACK_WEBHOOK_URL = data.aws_ssm_parameter.slack_webhook_url.value
      ENV               = var.env
    }
  }
}

data "aws_kms_key" "app_config_kms_key" {
  key_id = "alias/bcda-${var.env}-app-config-kms"
}


data "aws_iam_policy_document" "lambda_access" {
  statement {
    actions = [
      "ssm:GetParameter"
    ]
    resources = [
      "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/bcda/${var.env}/insights/*"
    ]
  }

  statement {
    actions = [
      "kms:Decrypt"
    ]
    resources = [
      data.aws_kms_key.app_config_kms_key.arn
    ]
  }

  statement {
    actions = [
      "kms:ListAliases"
    ]
    resources = [
      "*"
    ]
  }

  statement {
    actions = [
      "firehose:PutRecord"
    ]
    resources = [
      "arn:aws:firehose:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:deliverystream/*"
    ]
  }
}

resource "aws_iam_policy" "access_config" {
  name_prefix = "bcda-${var.env}-access-config-"
  description = "Allow lambda to access config"

  policy = data.aws_iam_policy_document.lambda_access.json

}

resource "aws_iam_role_policy_attachment" "access_config" {
  role       = aws_iam_role.insights_data_sampler_role.name
  policy_arn = aws_iam_policy.access_config.arn
}
