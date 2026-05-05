locals {
  env                     = "prod"
  static_site_domain_name = "bcda.cms.gov"
}

# S3 static site host bucket policy document
data "aws_iam_policy_document" "allow_cloudfront_access" {
  statement {
    sid    = "AllowCloudfrontAccess"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions = [
      "s3:GetObject",
      "s3:ListBucket"
    ]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"

      values = [
        module.cloudfront.distribution_arn,
      ]
    }

    resources = [
      module.s3_host.bucket_arn,
      "${module.s3_host.bucket_arn}/*",
    ]
  }
}

# S3 Bucket to host static site files
module "s3_host" {
  source                  = "../modules/s3_host"
  env                     = local.env
  versioning              = true
  tags                    = var.tags
  static_site_domain_name = local.static_site_domain_name
}

# Cloudfront distribution for static site
module "cloudfront" {
  source                  = "../modules/cloudfront"
  env                     = local.env
  tags                    = var.tags
  s3_origin_id            = module.s3_host.s3_origin_id
  static_site_domain_name = local.static_site_domain_name
}

# Allow Cloudfront to access objects in s3_host bucket
resource "aws_s3_bucket_policy" "allow_cloudfront_access" {
  bucket = module.s3_host.bucket_id
  policy = data.aws_iam_policy_document.allow_cloudfront_access.json
}