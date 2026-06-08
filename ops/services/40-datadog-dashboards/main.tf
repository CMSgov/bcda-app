terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~>6"
    }
    datadog = {
      source  = "DataDog/datadog"
      version = "~>4.4"
    }
  }
}

# Leverage per app- API and application keys that are managed by CDAP in services/datadog-cicd-keys
provider "datadog" {
  api_key = sensitive(module.platform.ssm.datadog.api_key.value)
  app_key = sensitive(module.platform.ssm.datadog.application_key.value)
  api_url = "https://api.ddog-gov.com"
}

module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=941672f97adfd8a19ce6533313302c4c74bac7a8"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app          = local.app
  env          = local.env
  root_module  = "https://github.com/CMSgov/bcda/tree/main/ops/services/65-dashboard"
  service      = local.service
  ssm_root_map = local.ssm_root_map
}

locals {
  default_tags = module.platform.default_tags
  env          = terraform.workspace
  service      = "dashboard"

  ssm_root_map = {
    common   = "/bcda/${local.env}/common"
    core     = "/bcda/${local.env}/core"
    accounts = "/bcda/mgmt/aws-account-numbers"
    splunk   = "/bcda/mgmt/splunk"
    datadog  = "/cdap/${local.env}/datadog/cicd/"
  }
}

module "datadog_dashboard" {
  source = "github.com/CMSgov/cdap//terraform/modules/datadog_dashboard?ref=945fbd644cc8d239bdf3f3a3a7241fb6066a0f55"

  app = local.app

  enable_default_widgets = {
    ecs    = true
    alb    = true
    aurora = true
    sns    = true
    sqs    = true
    lambda = true
    s3     = true
    apm    = true
  }

  widget_live_spans = {
    ecs    = "4h"
    alb    = "4h"
    aurora = "4h"
    sns    = "4h"
    sqs    = "4h"
    lambda = "1d"
    s3     = "1w"
    apm    = "1h"
  }

  custom_widgets = []
  runbook_url    = "https://confluence.cms.gov/spaces/BCDA/pages/421305405/BCDA+Alert+Runbooks"
}
