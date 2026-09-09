data "aws_iam_policy_document" "put_param" {
  statement {
    sid    = "SSMPutParameter"
    effect = "Allow"

    actions = ["ssm:PutParameter"]
    resources = [
      "arn:aws:ssm:${module.platform.primary_region.name}:${module.platform.account_id}:parameter/${local.app}/${local.env}/rotate-ssas-creds/*"
    ]
  }
}
