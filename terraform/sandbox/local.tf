locals {
  # VPC and Network related
  app_name              = "bcda"
  vpc_tag_name          = "bcda-east-${var.env}"
  ssm_cdap_cidr_name    = "/cdap/sensitive/mgmt-vpc/cidr"
  vpc_id                = data.aws_vpc.main.id
  app_cidr_block        = data.aws_vpc.main.cidr_block
  management_cidr_block = data.aws_ssm_parameter.cdap_cidr.value

  private_subnet_tag_use = "private"
  public_subnet_tag_use  = "public"

  # Security Group Names
  zscaler_private_name = "zscaler-private"

  # Route53 Zone
  local_zone_name = "bcda-${var.env}.local"

  # IAM related
  iam_path                       = "/delegatedadmin/developer/"
  instance_role_name             = "bcda-${var.env}-instance"
  instance_policy_name           = "bcda-${var.env}-instance"
  developer_boundary_policy_name = "developer-boundary-policy"

  # KMS related
  access_log_kms_description = "bcda-${var.env}-access-log-kms"
  access_log_kms_alias       = "alias/bcda-${var.env}-access-log-kms"
  app_config_kms_description = "bcda-${var.env}-app-config-kms"
  app_config_kms_alias       = "alias/bcda-${var.env}-app-config-kms"
  efs_kms_description        = "bcda-${var.env}-efs"
  efs_kms_alias              = "alias/bcda-${var.env}-efs"
  rds_kms_description        = "bcda-${var.env}-rds"
  rds_kms_alias              = "alias/bcda-${var.env}-rds"

  # EFS related
  efs_service_name  = "efs"
  efs_instance_name = "bcda-${var.env}-efs"

  # ALB related
  sg_compliance_attestation_url = "https://confluence.cms.gov/pages/viewpage.action?pageId=448009355"

  alb_sg_name                  = "bcda-app-lb-${var.env}"
  alb_sg_description           = "For BCDA app load balancer"
  alb_ingress_description      = "Ingress"
  alb_http_ingress_description = "Ingress HTTP"

  alb_name                             = "bcda-api-${var.env}-01"
  alb_tag_name                         = "bcda-${var.env}-app-01" # Tag for the ALB
  alb_idle_timeout                     = 60
  alb_deregistration_delay             = 120
  alb_health_check_interval            = 15
  health_check_start_period            = 10
  alb_health_check_path                = "/_health"
  alb_health_check_healthy_threshold   = 2
  alb_health_check_unhealthy_threshold = 5
  alb_health_check_timeout             = 5
  alb_certificate_arn                  = "arn:aws:acm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:certificate/d5e80222-d5ec-4bba-99fe-3849c8a18f84"

  # SSAS ALB related
  ssas_alb_sg_name        = "ssas-alb"
  ssas_alb_sg_description = "SSAS ALB security group"
  ssas_alb_name           = "bcda-ssas-${var.env}"
  ssas_alb_tag_name       = "bcda-${var.env}-ssas" # Tag for the SSAS ALB
  ssas_alb_idle_timeout   = 60
  ssas_certificate_arn    = "arn:aws:acm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:certificate/95d70efb-3dc6-485a-b8bd-2b93621b9091"

  # Security Groups (ASG)
  app_sg_name           = "bcda-api-${var.env}"
  app_sg_description    = "bcda api app security group"
  worker_sg_name        = "bcda-worker-${var.env}"
  worker_sg_description = "bcda worker security group"

  # Scaling
  api_desired_min    = 2
  api_desired_max    = 4
  ssas_desired_min   = 2
  ssas_desired_max   = 4
  worker_desired_min = 1
  worker_desired_max = 8

  # SNS related
  cloudwatch_alarms_topic_name          = "bcda-${var.env}-cloudwatch-alarms"
  cloudwatch_critical_alarms_topic_name = "bcda-${var.env}-cloudwatch-critical-alarms"

  # S3 related
  aco_credentials_bucket_name = "bcda-aco-credentials"

  # Ports
  https_port       = 443
  http_port        = 80
  admin_port       = 444  # Used for SSAS admin
  app_port         = 3000 # Default app port (HTTPS for ALB target)
  app_http_port    = 3001 # HTTP port for ALB target
  ssas_public_port = 3003
  ssas_admin_port  = 3004
  ssh_port         = 22
  postgres_port    = 5432

  # ECS
  container_name_api   = "api"
  ecs_task_def_cpu_api = 256
  ecs_task_def_mem_api = 512

  container_name_ssas   = "ssas"
  ecs_task_def_cpu_ssas = 256
  ecs_task_def_mem_ssas = 512

  ecs_task_def_cpu_worker = 2048
  ecs_task_def_mem_worker = 4096

  # User must match user(s) defined in dockerfile(s)
  app_user_uid = 1100
  app_user_gid = 1200

  # CloudWatch Alarm Default - Treat Missing Data
  alarm_elb_no_backend_treat_missing_data   = "notBreaching"
  alarm_elb_high_latency_treat_missing_data = "notBreaching"

  # Param store vars needed by each service
  param_names_api = [
    "ARCHIVE_THRESHOLD_HR",
    "AUTH_LOG",
    "BB_CHECK_CERT",
    "BB_CLIENT_CA_FILE",
    "BB_CLIENT_CERT_FILE",
    "BB_CLIENT_KEY_FILE",
    "BB_SERVER_LOCATION",
    "BCDA_API_CONFIG_PATH",
    "BCDA_BB_LOG",
    "BCDA_CA_FILE",
    "BCDA_ENABLE_NEW_GROUP",
    "BCDA_ERROR_LOG",
    "BCDA_REQUEST_LOG",
    "BCDA_SSAS_CLIENT_ID",
    "BCDA_SSAS_SECRET",
    "BCDA_TLS_CERT",
    "BCDA_TLS_KEY",
    "CCLF_CUTOFF_DATE_DAYS",
    "CLIENT_RETRY_AFTER_IN_SECONDS",
    "CONFIG_BUCKET",
    "DATABASE_URL",
    "DEBUG",
    "DEPLOYMENT_TARGET",
    "FHIR_ARCHIVE_DIR",
    "FHIR_PAYLOAD_DIR",
    "FHIR_STAGING_DIR",
    "JWT_PRIVATE_KEY_FILE",
    "JWT_PUBLIC_KEY_FILE",
    "PRIORITY_ACO_IDS",
    "PRIORITY_ACO_REG_EX",
    "RUNOUT_CLAIM_THRU_DATE",
    "RUNOUT_CUTOFF_DATE_DAYS",
    "SSAS_PUBLIC_URL",
    "SSAS_TIMEOUT_MS",
    "SSAS_URL",
    "SSAS_USE_TLS",
    "V3_BB_SERVER_LOCATION",
    "VERSION_2_ENDPOINT_ACTIVE",
    "VERSION_3_ENDPOINT_ACTIVE",
    "USER_GUIDE_LOC"
  ]

  param_names_ssas = [
    "BCDA_TLS_CERT",
    "BCDA_TLS_KEY",
    "CONFIG_BUCKET",
    "DATABASE_URL",
    "DEBUG",
    "DEPLOYMENT_TARGET",
    "GOPATH",
    "SSAS_ADMIN_SIGNING_KEY_PATH",
    "SSAS_CRED_EXPIRATION_DAYS",
    "SSAS_DEFAULT_SYSTEM_SCOPE",
    "SSAS_HASH_ITERATIONS",
    "SSAS_HASH_KEY_LENGTH",
    "SSAS_HASH_SALT_SIZE",
    "SSAS_IDLE_TIMEOUT",
    "SSAS_LOG",
    "SSAS_MFA_TOKEN_TIMEOUT_MINUTES",
    "SSAS_PUBLIC_SIGNING_KEY_PATH",
    "SSAS_PUBLIC_URL",
    "SSAS_READ_TIMEOUT",
    "SSAS_TOKEN_DENYLIST_CACHE_CLEANUP_MINUTES",
    "SSAS_TOKEN_DENYLIST_CACHE_TIMEOUT_MINUTES",
    "SSAS_WRITE_TIMEOUT"
  ]

  param_names_worker = [
    "BB_CHECK_CERT",
    "BB_CLIENT_CA_FILE",
    "BB_CLIENT_CERT_FILE",
    "BB_CLIENT_KEY_FILE",
    "BB_SERVER_LOCATION",
    "BB_TIMEOUT_MS",
    "BCDA_BB_LOG",
    "BCDA_WORKER_CONFIG_PATH",
    "BCDA_WORKER_ERROR_LOG",
    "COMPRESSION_LEVEL",
    "CONFIG_BUCKET",
    "DATABASE_URL",
    "DEPLOYMENT_TARGET",
    "EXPORT_FAIL_PCT",
    "FHIR_ARCHIVE_DIR",
    "FHIR_PAYLOAD_DIR",
    "FHIR_STAGING_DIR",
    "SLACK_TOKEN",
    "V3_BB_SERVER_LOCATION",
    "WORKER_HEALTH_INT_SEC",
    "WORKER_HEALTH_LOG",
    "WORKER_POOL_SIZE"
  ]
}
