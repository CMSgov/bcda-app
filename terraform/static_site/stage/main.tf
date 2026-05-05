locals {
  env                                  = "stage"
  static_site_domain_name              = "stage.bcda.cms.gov"
  static_site_distribution_domain_name = "d3obyxkdm48edr.cloudfront.net"
  cloudmgmt_addresses = [
    "13.217.185.142/32", # cdap-east-mgmt-nat-gateway-a
    "3.226.187.84/32", # cdap-east-mgmt-nat-gateway-b
    "52.55.143.28/32", # cdap-east-mgmt-nat-gateway-c
  ]
  newrelic_addresses = [
    "18.217.88.49/32",
    "18.221.231.23/32",
    "18.217.159.174/32",
    "3.145.224.0/24",
    "3.145.225.0/25",
    "3.145.234.0/24",
    "3.130.159.252/32",
    "3.13.7.11/32",
    "3.130.155.242/32"
  ]
  zscaler_addresses = [
    "65.213.206.0/24",
    "208.250.57.0/24",
  ]
}

# CMS VPN Source / Cloud Services IPs
resource "aws_wafv2_ip_set" "cms_cloud_services" {
  name               = "BCDA_CMS_Cloud_Services_IP_Set"
  description        = "IP set for CMS VPN Source / Cloud Services IPs"
  scope              = "CLOUDFRONT"
  ip_address_version = "IPV4"
  addresses          = concat(local.cloudmgmt_addresses, local.newrelic_addresses, local.zscaler_addresses)
}

# Note: this Rule Group must be manually attached to the WAF ACL after deployment
resource "aws_wafv2_rule_group" "static_site_staging_ip_blocking" {

  name        = "BCDAStaticSiteStagingIPBlocking"
  description = "Rule Group to block non VPN traffic to the BCDA static site in the stage environment"
  scope       = "CLOUDFRONT"
  capacity    = 25

  rule {
    name     = "BlockNonVPNTraffic"
    priority = 1

    action {
      block {}
    }

    statement {
      and_statement {
        # Request Host header matches static site domain in Stage
        statement {
          or_statement {
            statement {
              byte_match_statement {
                field_to_match {
                  single_header {
                    name = "host"
                  }
                }
                positional_constraint = "CONTAINS"
                search_string         = local.static_site_domain_name
                text_transformation {
                  priority = 1
                  type     = "NONE"
                }
              }
            }
            statement {
              byte_match_statement {
                field_to_match {
                  single_header {
                    name = "host"
                  }
                }
                positional_constraint = "CONTAINS"
                search_string         = local.static_site_distribution_domain_name
                text_transformation {
                  priority = 1
                  type     = "NONE"
                }
              }
            }
          }
        }

        # Source IP does NOT match CMS VPN IP Set
        statement {
          not_statement {
            statement {
              ip_set_reference_statement {
                arn = aws_wafv2_ip_set.cms_cloud_services.arn
              }
            }
          }
        }
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "BlockNonVPNTrafficRule"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "BlockNonVPNTrafficRuleGroup"
    sampled_requests_enabled   = true
  }
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
  source = "../modules/s3_host"

  env                     = local.env
  versioning              = true
  tags                    = var.tags
  static_site_domain_name = local.static_site_domain_name
}

# Cloudfront distribution for static site
module "cloudfront" {
  source = "../modules/cloudfront"

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
