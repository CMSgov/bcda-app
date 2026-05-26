locals {
  app        = "bcda"
  service    = "datadog-dashboards"
}

module "datadog_dashboard" {
  source = "github.com/CMSgov/cdap//terraform/modules/datadog_dashboard"


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
  runbook_url    = "https://definerunbook.cdap.internal.cms.gov" #FIXME to provide an actual runbook
}

module "platform" {
  source = "github.com/CMSgov/cdap//terraform/modules/platform?ref=941672f97adfd8a19ce6533313302c4c74bac7a8"

  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = local.app
  env         = var.env
  root_module = "https://github.com/CMSgov/bcda-app/tree/main/ops/services/535-datadog-dashboards"
  service     = local.service
}

