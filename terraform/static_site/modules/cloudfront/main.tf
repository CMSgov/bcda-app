# Fetch the region-agnostic WAF ACL meant specifically for Cloudfront
data "aws_wafv2_web_acl" "cms_cloudfront_waf" {
  name  = "SamQuickACLEnforcingV2"
  scope = "CLOUDFRONT"
}

resource "aws_cloudfront_origin_access_control" "s3_origin" {
  name                              = "${var.static_site_domain_name}-s3-origin"
  description                       = "bcda static site s3 origin"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_function" "redirects" {
  name    = "redesign-redirects"
  runtime = "cloudfront-js-1.0"
  comment = "Handle cool URIs and redirects for the redesign"
  code    = file("${path.module}/redirects-function.js")
}

resource "aws_cloudfront_distribution" "static_site_distribution" {
  origin {
    domain_name              = "${var.s3_origin_id}.s3.us-east-1.amazonaws.com"
    origin_id                = var.s3_origin_id
    origin_access_control_id = aws_cloudfront_origin_access_control.s3_origin.id
  }

  aliases = ["${var.static_site_domain_name}"]

  web_acl_id          = data.aws_wafv2_web_acl.cms_cloudfront_waf.arn
  enabled             = true
  default_root_object = "index.html"
  is_ipv6_enabled     = true
  http_version        = "http2and3"


  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = var.s3_origin_id
    compress               = true
    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.redirects.arn
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "whitelist"
      locations        = ["US", "CA", "GB"]
    }
  }

  viewer_certificate {
    acm_certificate_arn      = data.aws_acm_certificate.issued.arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  custom_error_response {
    error_caching_min_ttl = 10
    error_code            = 403
    response_code         = 404
    response_page_path    = "/404.html"
  }

  custom_error_response {
    error_caching_min_ttl = 10
    error_code            = 404
    response_code         = 404
    response_page_path    = "/404.html"
  }

}

data "aws_acm_certificate" "issued" {
  domain   = var.static_site_domain_name
  statuses = ["ISSUED"]
}
