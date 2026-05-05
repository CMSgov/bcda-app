data "aws_ecr_lifecycle_policy_document" "this" {
  rule {
    priority    = 1
    description = "Keep the last 3 release images"

    selection {
      tag_status      = "tagged"
      tag_prefix_list = ["r"]
      count_type      = "imageCountMoreThan"
      count_number    = 3
    }
  }

  rule {
    priority    = 2
    description = "Keep the last 3 temp images"

    selection {
      tag_status      = "tagged"
      tag_prefix_list = ["temp-"]
      count_type      = "imageCountMoreThan"
      count_number    = 3
    }
  }

  rule {
    priority    = 3
    description = "Drop untagged images after 30 days"

    selection {
      tag_status   = "untagged"
      count_type   = "sinceImagePushed"
      count_unit   = "days"
      count_number = 30
    }
  }
}

resource "aws_ecr_repository" "this" {
  name = var.name

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = var.platform.kms_alias_primary.target_key_arn
  }
}

resource "aws_ecr_lifecycle_policy" "this" {
  repository = aws_ecr_repository.this.name
  policy     = data.aws_ecr_lifecycle_policy_document.this.json
}
