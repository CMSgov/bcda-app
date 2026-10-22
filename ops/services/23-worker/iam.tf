data "aws_iam_policy_document" "worker_task" {
  statement {
    sid     = "AllowKMSAppConfig"
    actions = ["kms:Decrypt", "kms:Encrypt", "kms:GenerateDataKey", "kms:ListAliases"]
    resources = [
      data.aws_kms_key.app_config_kms_key.arn
    ]
  }

  statement {
    sid     = "AllowConfigBucketRead"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${data.aws_ssm_parameter.params_worker["CONFIG_BUCKET"].value}",
      "arn:aws:s3:::${data.aws_ssm_parameter.params_worker["CONFIG_BUCKET"].value}/*"
    ]
  }

  statement {
    sid     = "AllowACOCredsBucketWrite"
    actions = ["s3:PutObject"]
    resources = [
      "arn:aws:s3:::bcda-aco-credentials/*"
    ]
  }
}

resource "aws_iam_policy" "worker_task" {
  name   = "bcda-${module.platform.env}-worker-task"
  path   = "/delegatedadmin/developer/"
  policy = data.aws_iam_policy_document.worker_task.json
}
