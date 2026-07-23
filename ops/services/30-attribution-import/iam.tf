locals {
  assume_role_arns = compact(split(",", module.platform.ssm.attribution-import.delivery_role_arns.value))
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

  statement {
    actions = ["sts:AssumeRoleWithWebIdentity", "sts:TagSession"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "${local.provider_domain}:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "${local.provider_domain}:sub"
      values   = ["repo:CMSgov/bcda-app:*"]
    }
  }
}

data "aws_iam_policy_document" "bucket_upload" {
  statement {
    sid    = "AllowS3Upload"
    effect = "Allow"

    actions = [
      "s3:PutObject",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts"
    ]
    resources = [
      "${module.attribution-import_file_bucket.arn}/*"
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
  name = "bcda-${local.env}-attribution-import-delivery"

  assume_role_policy = data.aws_iam_policy_document.delivery_assume_role.json
}

resource "aws_iam_role_policy" "bucket_upload" {
  name   = "attribution-import-bucket-upload"
  role   = aws_iam_role.delivery.id
  policy = data.aws_iam_policy_document.bucket_upload.json
}

data "aws_iam_policy_document" "bucket_manage" {
  statement {
    sid    = "ListBucket"
    effect = "Allow"

    actions = [
      "s3:ListBucket",
      "s3:GetBucketLocation",
    ]

    resources = [
      module.attribution-import_file_bucket.arn,
    ]
  }

  statement {
    sid    = "ReadDeleteObjects"
    effect = "Allow"

    actions = [
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:GetObjectTagging",
      "s3:DeleteObject",
      "s3:DeleteObjectVersion",
    ]

    resources = [
      "${module.attribution-import_file_bucket.arn}/*"
    ]
  }
}

data "aws_iam_policy_document" "bucket_sqs" {
  statement {
    sid = "SqsReceiveDeleteMessages"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:GetQueueAttributes",
      "sqs:DeleteMessage",
    ]
    resources = [module.attribution_import_queue.arn]
  }
}