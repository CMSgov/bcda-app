locals {

  ## Evaluates config/defaults.yml and overwrites values with those from config/${var.env}.yml for each
  ## variable/key type. Creates a hierarchy of defaults, so the modules/datadog_monitors defaults are
  ## the least prioritized, followed by config/defaults.yml, followed by the environment specific settings.
  env        = local.parent_env
  defaults   = yamldecode(file("config/defaults.yml"))
  env_config = yamldecode(file("config/${local.env}.yml"))

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

  health_check_config = merge(
    lookup(local.defaults, "health_check", {}),
    lookup(local.env_config, "health_check", {})
  )
  has_health_check = try(local.health_check_config.enabled, false) && length(try(local.health_check_config.validates_json_path, [])) > 0
}

# Use platform module to derive datadog keys via ssm_root_map
# Can be replaced with direct data lookups
module "platform" {
  source    = "github.com/CMSgov/cdap//terraform/modules/platform?ref=ea161d6a00e729690a495d30c4d57d0d2990d0a6"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app          = "bcda"
  env          = local.env
  root_module  = "https://github.com/CMSgov/bcda/tree/main/terraform/services/${basename(abspath(path.module))}/"
  service      = replace(basename(abspath(path.module)), "/^[0-9]+-/", "")
  ssm_root_map = { datadog = "/bcda/${local.env}/datadog/cicd/" }
}

# Synthetics Tests

module "datadog_synthetics" {
  count  = local.has_health_check ? 1 : 0
  source = "github.com/CMSgov/cdap//terraform/modules/datadog_synthetics?ref=ea161d6a00e729690a495d30c4d57d0d2990d0a6"

  app                  = "bcda"
  env                  = local.env
  notify               = module.common_datadog_monitors.notify
  min_failure_duration = lookup(local.health_check_config, "min_failure_duration", 0)

  tests = [
    {
      name    = "Health Check"
      type    = "api"
      subtype = "http"
      status  = "live"

      request_definition = {
        method = "GET"
        url    = local.health_check_config.url
      }

      assertions = concat(
        [
          {
            type     = "responseTime"
            operator = "lessThan"
            target   = tostring(lookup(local.health_check_config, "max_response_time_ms", 1000))
          },
          {
            type     = "statusCode"
            operator = "is"
            target   = "200"
          }
        ],
        [
          for field in try(local.health_check_config.validates_json_path, []) : {
            type     = "body"
            operator = "validatesJSONPath"
            targetjsonpath = {
              jsonpath    = "$.${field}"
              operator    = "contains"
              targetvalue = "ok"
            }
          }
        ]
      )

      tick_every           = lookup(local.health_check_config, "tick_every", 1800)
      min_failure_duration = lookup(local.health_check_config, "min_failure_duration", 3600)
    }
  ]
}

# Common Monitors

module "common_datadog_monitors" {
  source = "github.com/CMSgov/cdap//terraform/modules/datadog_monitors?ref=ea161d6a00e729690a495d30c4d57d0d2990d0a6"

  app            = "bcda"
  env            = local.env
  monitor_config = local.monitor_config
}
