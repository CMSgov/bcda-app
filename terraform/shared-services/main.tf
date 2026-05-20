provider "aws" {}

/* ---------- ACO CREDENTIALS S3 BUCKET -------------- */
resource "aws_kms_key" "aco_creds_kms_key" {
  description             = "bcda-aco-creds-kms"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "aco_creds_kms_alias" {
  name          = "alias/bcda-aco-creds-kms"
  target_key_id = aws_kms_key.aco_creds_kms_key.key_id
}

resource "aws_s3_bucket" "aco_creds" {
  bucket = "bcda-aco-credentials"

  # Explicitly disabling versioning since these credentials are intended to be short-lived.
  #ts:skip=AWS.S3Bucket.IAM.High.0370 Credentials are intended to be short lived so the bucket which stores these credentials should not be versioned.
  versioning {
    enabled = false
  }

  lifecycle_rule {
    id      = "Clean-up-Objects-dev"
    enabled = true

    prefix = "dev/"

    expiration {
      days = 30
    }
  }

  lifecycle_rule {
    id      = "Clean-up-Objects-test"
    enabled = true

    prefix = "test/"

    expiration {
      days = 30
    }
  }

  lifecycle_rule {
    id      = "Clean-up-Objects-sandbox"
    enabled = true

    prefix = "sandbox/"

    expiration {
      days = 30
    }
  }

  lifecycle_rule {
    id      = "Clean-up-Objects-prod"
    enabled = true

    prefix = "prod/"

    expiration {
      days = 30
    }
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.aco_creds_kms_key.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Id": "bcda-PutObjPolicy",
    "Statement": [
        {
            "Sid": "DenyIncorrectEncryptionHeader",
            "Effect": "Deny",
            "Principal": "*",
            "Action": "s3:PutObject",
            "Resource": "arn:aws:s3:::bcda-aco-credentials/*",
            "Condition": {
                "StringNotEquals": {
                    "s3:x-amz-server-side-encryption": "aws:kms"
                }
            }
        },
        {
            "Sid": "DenyUnEncryptedObjectUploads",
            "Effect": "Deny",
            "Principal": "*",
            "Action": "s3:PutObject",
            "Resource": "arn:aws:s3:::bcda-aco-credentials/*",
            "Condition": {
                "Null": {
                    "s3:x-amz-server-side-encryption": "true"
                }
            }
        },
        {
            "Sid": "AllowSSLRequestsOnly",
            "Effect": "Deny",
            "Principal": "*",
            "Action": "s3:*",
            "Resource": [
                "arn:aws:s3:::bcda-aco-credentials",
                "arn:aws:s3:::bcda-aco-credentials/*"
            ],
            "Condition": {
                "Bool": {
                    "aws:SecureTransport": "false"
                }
            }
        }
    ]
}
EOF

  tags = {
    Name                                = "bcda-aco-credentials",
    "cms-cloud-exempt:public-s3-bucket" = "CLDSPT-8353"
  }
}

resource "aws_s3_bucket_public_access_block" "public-access-block" {
  bucket = aws_s3_bucket.aco_creds.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

