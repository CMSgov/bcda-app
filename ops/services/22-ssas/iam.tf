data "aws_iam_policy_document" "ssas_task" {
  statement {
    sid     = "AllowKMSAppConfig"
    actions = ["kms:Encrypt", "kms:GenerateDataKey", "kms:ListAliases"]
    resources = [
      "arn:aws:kms:*:${module.platform.account_id}:key/${data.aws_kms_key.app_config_kms_key.id}"
    ]
  }

  statement {
    sid     = "AllowConfigBucketRead"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${data.aws_ssm_parameter.config_bucket.value}",
      "arn:aws:s3:::${data.aws_ssm_parameter.config_bucket.value}/*"
    ]
  }

  statement {
    sid     = "AllowSSMParamsByPath"
    actions = ["ssm:GetParametersByPath"]
    resources = [
      "arn:aws:ssm:${module.platform.primary_region.name}:${module.platform.account_id}:parameter/bcda/${module.platform.env}/*"
    ]
  }

  statement {
    sid     = "AllowSlackToken"
    actions = ["ssm:GetParameter"]
    resources = [
      "arn:aws:ssm:${module.platform.primary_region.name}:${module.platform.account_id}:parameter/slack/token/workflow-alerts"
    ]
  }
}

resource "aws_iam_policy" "ssas_task" {
  name   = "bcda-${module.platform.env}-ssas-task"
  policy = data.aws_iam_policy_document.ssas_task.json
}
