locals {
  assume_role_arns = split(",", module.platform.ssm.attribution-import.delivery_role_arns.value)
}

data "aws_iam_role" "admin" {
  name = "ct-ado-bcda-application-admin"
}

data "aws_iam_role" "dasg_admin" {
  name = "ct-ado-dasg-application-admin"
}

data "aws_iam_policy_document" "delivery_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type = "AWS"
      identifiers = concat(
        [
          data.aws_iam_role.admin.arn,
          data.aws_iam_role.dasg_admin.arn,
        ],
        local.assume_role_arns
      )
    }
  }
}

data "aws_iam_policy_document" "attribution-import_bucket_upload" {
  statement {
    sid    = "AllowS3Upload"
    effect = "Allow"

    actions = [
      "s3:PutObject",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts"
    ]
    resources = [
      "${module.attribution-import_file_bucket.arn}/bfdeft01/bcda/in/*",
      "${module.attribution-import_file_bucket.arn}/bfdeft01/bcda/in/test/*"
    ]
  }
  statement {
    sid    = "AllowKMSEncryption"
    effect = "Allow"

    actions = [
      "kms:GenerateDataKey",
      "kms:Encrypt",
      "kms:Decrypt",
    ]
    resources = [aws_kms_key.attribution-import_bucket.arn]
  }
}

resource "aws_iam_role" "delivery" {
  name = "bcda-${var.env}-attribution-import-delivery"

  assume_role_policy = data.aws_iam_policy_document.delivery_assume_role.json
}

resource "aws_iam_role_policy" "attribution-import_bucket_upload" {
  name   = "attribution-import-bucket-upload"
  role   = aws_iam_role.delivery.id
  policy = data.aws_iam_policy_document.attribution-import_bucket_upload.json
}
