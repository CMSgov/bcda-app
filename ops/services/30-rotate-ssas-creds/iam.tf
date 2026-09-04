data "aws_iam_policy_document" "put_param" {
  statement {
    sid    = "SSMPutParameter"
    effect = "Allow"

    actions = ["ssm:PutParameter"]
    resources = [
      "arn:aws:ssm:us-east-1:539247469933:parameter/${local.app}/${local.env}/creds/*"
    ]
  }
}
