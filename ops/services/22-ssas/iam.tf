data "aws_iam_policy_document" "ssas_task" {
  statement {
    sid     = "AllowKMSAppConfig"
    actions = ["kms:Encrypt", "kms:GenerateDataKey", "kms:ListAliases"]
    resources = [
      "arn:aws:kms:*:${module.platform.account_id}:key/${module.platform.app_config_kms_key_id}"
    ]
  }

  statement {
    sid     = "AllowConfigBucketRead"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${module.platform.config_bucket_ssas}",
      "arn:aws:s3:::${module.platform.config_bucket_ssas}/*"
    ]
  }

  statement {
    sid     = "AllowSSMParamsByPath"
    actions = ["ssm:GetParametersByPath"]
    resources = [
      "arn:aws:ssm:${module.platform.region}:${module.platform.account_id}:parameter/bcda/${module.platform.env}/*"
    ]
  }

  statement {
    sid     = "AllowSlackToken"
    actions = ["ssm:GetParameter"]
    resources = [
      "arn:aws:ssm:${module.platform.region}:${module.platform.account_id}:parameter/slack/token/workflow-alerts"
    ]
  }
}

resource "aws_iam_policy" "ssas_task" {
  name   = "bcda-${module.platform.env}-ssas-task"
  path   = module.platform.iam_path
  policy = data.aws_iam_policy_document.ssas_task.json
}
