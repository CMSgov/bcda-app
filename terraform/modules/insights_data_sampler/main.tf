locals {
  tags = { business = "OEDA", application = "bfd-insights", project = "bcda" }
}

data "aws_region" "current" {}

data "aws_caller_identity" "current" {}

# cloudwatch event
resource "aws_cloudwatch_event_rule" "main" {
  name                = "bfd-insights-${var.database}-${var.env}-${var.name}"
  description         = var.description
  schedule_expression = var.schedule
}

resource "aws_cloudwatch_event_target" "main" {
  rule = aws_cloudwatch_event_rule.main.name
  arn  = var.lambda_arn

  input = jsonencode(
    {
      name                   = aws_cloudwatch_event_rule.main.name
      query                  = var.query
      delivery_stream        = "bfd-insights-${var.database}-${var.env}-${var.name}"
      db_conn_string_env_var = var.db_conn_string_env_var
    }
  )
}

resource "aws_lambda_permission" "main" {
  statement_id_prefix = "InsightsDataSampler-AllowExecutionFromCloudWatch-"
  action              = "lambda:InvokeFunction"
  function_name       = "insights_data_sampler_${var.env}"
  principal           = "events.amazonaws.com"
  source_arn          = aws_cloudwatch_event_rule.main.arn
}

resource "aws_kms_key" "cloudwatch_kms_key" {
  description             = "bfd-insights-${var.database}-${var.env}-${var.name}-cloudwatch-key"
  deletion_window_in_days = 10
  enable_key_rotation     = true

  policy = <<EOF
{
   "Version":"2012-10-17",
   "Id":"bfd-insights-${var.database}-${var.env}-${var.name}-cloudwatch-key-policy",
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
              "kms:EncryptionContext:aws:logs:arn": "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/bfd-insights-${var.database}-${var.env}-${var.name}"
           }
         }
      }
   ]
}
EOF

}

resource "aws_kms_alias" "cloudwatch_kms_key_alias" {
  name          = "alias/bfd-insights-${var.database}-${var.env}-${var.name}-key"
  target_key_id = aws_kms_key.cloudwatch_kms_key.key_id
}

resource "aws_cloudwatch_log_group" "firehose_log_group" {
  name              = "/aws/kinesisfirehose/bfd-insights-${var.database}-${var.env}-${var.name}"
  retention_in_days = 14
  kms_key_id        = aws_kms_key.cloudwatch_kms_key.arn
}

resource "aws_cloudwatch_log_stream" "firehose_log_stream" {
  name           = "S3Delivery"
  log_group_name = aws_cloudwatch_log_group.firehose_log_group.name
}
