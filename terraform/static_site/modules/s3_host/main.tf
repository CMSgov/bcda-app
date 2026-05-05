# Private S3 bucket to host the BCDA static site build files

resource "aws_s3_bucket" "s3_host" {
  bucket_prefix = var.static_site_domain_name
  tags          = merge({ Name = var.static_site_domain_name }, var.tags)
}

resource "aws_s3_bucket_ownership_controls" "bucket_owner" {
  bucket = aws_s3_bucket.s3_host.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_versioning" "versions" {
  bucket = aws_s3_bucket.s3_host.id
  versioning_configuration {
    status     = "Enabled"
    mfa_delete = "Disabled"
  }
}

data "aws_caller_identity" "current" {}

resource "aws_s3_bucket_logging" "s3_logs" {
  bucket        = aws_s3_bucket.s3_host.id
  target_bucket = "cms-cloud-${data.aws_caller_identity.current.account_id}-us-east-1"
  target_prefix = "AWSLogs/BCDA-S3-Access/${data.aws_caller_identity.current.account_id}/${var.static_site_domain_name}-s3-access-logs/"
}

resource "aws_ssm_parameter" "static_site_bucket" {
  name        = "/bcda/${var.env}/static_site"
  description = "Static site bucket"
  type        = "String"
  value       = aws_s3_bucket.s3_host.bucket
  tags = {
    environment = var.env
  }
}

resource "aws_s3_bucket_acl" "s3_host_acl" {
  depends_on = [aws_s3_bucket_ownership_controls.bucket_owner]
  bucket = aws_s3_bucket.s3_host.id
  acl    = "private"
}

resource "aws_s3_bucket_lifecycle_configuration" "this" {
  bucket = aws_s3_bucket.s3_host.id

  rule {
    id     = "noncurrent-ia-tagged"
    status = "Enabled"

    filter {
      tag {
        key   = "lifecycle-transition"
        value = "ia"
      }
    }

    noncurrent_version_transition {
      noncurrent_days = 30
      storage_class   = "STANDARD_IA"
    }
  }

  rule {
    id     = "cleanup-multipart"
    status = "Enabled"

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}
