locals {

  ## Evaluates config/defaults.yml and overwrites values with those from config/${var.env}.yml for each
  ## variable/key type. Creates a hierarchy of defaults, so the modules/datadog_monitors defaults are
  ## the least prioritized, followed by config/defaults.yml, followed by the environment specific settings.

  defaults   = yamldecode(file("config/defaults.yml"))
  env_config = yamldecode(file("config/${var.env}.yml"))

  shadow_mode = lookup(local.env_config, "shadow_mode", local.defaults.shadow_mode)

  # map-typed keys
  monitor_config = merge(
    { for key in keys(local.defaults) : key => merge(
      lookup(local.defaults, key, {}),
      lookup(local.env_config, key, {})
      ) if can(keys(local.defaults[key])) # only process map-typed keys
    },
    { shadow_mode = local.shadow_mode }
  )

  # handles a case where the notifications are null
  _env_channels = try(local.env_config.notifications.channels, null)

  # always use the notification channels set up in the defaults, and adds those from the environment
  notify = join(" ", concat(
    local.defaults.notifications.channels,
    local._env_channels != null ? local._env_channels : []
  ))
}

###################
# Common Monitors #
###################

module "common_datadog_monitors" {
  source = "../../modules/datadog_monitors"

  app            = "cdap"
  env            = var.env
  monitor_config = local.monitor_config
  notify         = local.notify
}

# Use platform module to derive datadog keys via ssm_root_map
# Can be replaced with direct data lookups 
module "platform" {
  source    = "../../modules/platform"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app          = "cdap"
  env          = var.env
  root_module  = "https://github.com/CMSgov/cdap/tree/main/terraform/services/${basename(abspath(path.module))}/"
  service      = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")
  ssm_root_map = { datadog = "/cdap/${var.env}/datadog/cicd/" }
}


##########################
# CDAP Specific Monitors #
##########################

locals {
  codebuild_repos = [
    "ab2d",
    "ab2d-website",
    "bcda-app",
    "bcda-ssas-app",
    "bcda-static-site",
    "cdap",
    "dpc-app",
    "dpc-ops",
    "dpc-static-site",
  ]
}

resource "datadog_monitor" "codebuild_failed_builds" {
  for_each = toset(local.codebuild_repos)

  name    = "[${upper(module.platform.account_env_suffix)}] [${each.key}]  CodeBuild — Failed Builds"
  type    = "metric alert"
  message = "CodeBuild project ${each.key}-${module.platform.account_env_suffix} has failing builds. ${local.notify}"

  query = "sum(last_30m):sum:aws.codebuild.failed_builds{projectname:${each.key}-${module.platform.account_env_suffix}}> 1"

  monitor_thresholds {
    critical = 1
  }

  tags = [
    "application:${each.key}",
    "environment:${var.env}",
    "managed-by:tofu",
  ]
}

resource "datadog_monitor" "codebuild_queue_backup" {
  for_each = toset(local.codebuild_repos)

  name    = "[${upper(module.platform.account_env_suffix)}] [${each.key}] CodeBuild — Builds Backing Up in Queue"
  type    = "metric alert"
  message = "CodeBuild project ${each.key}-${module.platform.account_env_suffix} builds are queuing — runners may be unavailable. ${local.notify}"

  query = "avg(last_10m):avg:aws.codebuild.queued_duration{projectname:${each.key}-${module.platform.account_env_suffix}} > 120"

  monitor_thresholds {
    critical = 120
    warning  = 72
  }

  tags = [
    "application:${each.key}",
    "environment:${var.env}",
    "managed-by:tofu",
  ]
}
