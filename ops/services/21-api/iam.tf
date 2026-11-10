data "aws_iam_policy_document" "api_task" {
  statement {
    sid     = "AllowKMSAppConfig"
    actions = ["kms:Encrypt", "kms:Decrypt", "kms:GenerateDataKey", "kms:ListAliases"]
    resources = [
      data.aws_kms_key.app_config_kms_key.arn
    ]
  }

  statement {
    sid     = "AllowConfigBucketRead"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${data.aws_ssm_parameter.params_api["CONFIG_BUCKET"].value}",
      "arn:aws:s3:::${data.aws_ssm_parameter.params_api["CONFIG_BUCKET"].value}/*"
    ]
  }

  statement {
    sid     = "AllowACOCredsBucketWrite"
    actions = ["s3:PutObject"]
    resources = [
      "arn:aws:s3:::bcda-aco-credentials/*"
    ]
  }

  statement {
    sid     = "AllowSSMParamsByPath"
    actions = ["ssm:GetParametersByPath"]
    resources = [
      "arn:aws:ssm:${module.platform.primary_region.name}:${module.platform.aws_caller_identity.account_id}:parameter/bcda/${module.platform.env}/*"
    ]
  }
}

resource "aws_iam_policy" "api_task" {
  name   = "bcda-${module.platform.env}-api-task"
  path   = "/delegatedadmin/developer/"
  policy = data.aws_iam_policy_document.api_task.json
}
