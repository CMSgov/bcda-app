data "aws_region" "current" {}

data "aws_vpc" "main" {
  filter {
    name   = "tag:Name"
    values = [local.vpc_tag_name]
  }
}

data "aws_ssm_parameter" "cdap_cidr" {
  name = local.ssm_cdap_cidr_name
}

data "aws_subnets" "private_subnets" {
  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
  filter {
    name   = "tag:use"
    values = [local.private_subnet_tag_use]
  }
}

data "aws_subnets" "public_subnets" {
  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
  filter {
    name   = "tag:use"
    values = [local.public_subnet_tag_use]
  }
}

data "aws_subnets" "all_subnets" {}

data "aws_elb_service_account" "main" {
}

data "aws_caller_identity" "current" {
}

data "aws_security_group" "zscaler_private" {
  name = local.zscaler_private_name
  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
}

data "aws_security_group" "db" {
  name = "bcda-${var.env}-db"
}

/* ---- Route 53 Zone ---- */
resource "aws_route53_zone" "local_zone" {
  name = local.local_zone_name

  vpc {
    vpc_id = data.aws_vpc.main.id
  }
}

/* ---- IAM configs ---- */
data "aws_iam_policy" "developer_boundary" {
  name = local.developer_boundary_policy_name
}

resource "aws_iam_role" "instance" {
  name = local.instance_role_name
  path = local.iam_path

  permissions_boundary = data.aws_iam_policy.developer_boundary.arn

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": [
          "ecs-tasks.amazonaws.com"
        ]
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

}

resource "aws_iam_policy" "instance" {
  name = local.instance_policy_name
  path = local.iam_path

  description = "Allow instances to bootstrap themselves"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1538172026254",
            "Action": [
              "elasticfilesystem:DescribeFileSystems"
            ],
            "Effect": "Allow",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
              "kms:Decrypt",
              "kms:GenerateDataKey"
            ],
            "Resource": "arn:aws:kms:*:${data.aws_caller_identity.current.account_id}:key/${aws_kms_key.app_config_kms_key.key_id}"
         },
         {
            "Effect": "Allow",
            "Action": [
              "kms:Encrypt",
			  "kms:GenerateDataKey"
            ],
            "Resource": [
			"arn:aws:kms:*:${data.aws_caller_identity.current.account_id}:key/${aws_kms_key.app_config_kms_key.key_id}"
			]
         },
         {
            "Effect": "Allow",
            "Action": [
			  "kms:ListAliases"
            ],
            "Resource": [
			  "*"
			]
		 },
		 {
            "Effect": "Allow",
            "Action": [
			  "s3:PutObject"
			],
            "Resource": [
			  "arn:aws:s3:::${local.aco_credentials_bucket_name}/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
              "cloudwatch:PutMetricData"
                              ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
              "logs:CreateLogGroup",
              "logs:CreateLogStream",
              "logs:PutLogEvents"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
              "firehose:PutRecord"
                              ],
            "Resource": "arn:aws:firehose:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:deliverystream/*"
        },
        {
          "Effect": "Allow",
          "Action": "ssm:GetParametersByPath",
          "Resource": "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/${local.app_name}/${var.env}/*"
        },
        {
          "Effect": "Allow",
          "Action": "ssm:GetParameter",
          "Resource": "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/slack/token/workflow-alerts"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ssm:DescribeAssociation",
                "ssm:GetDeployablePatchSnapshotForInstance",
                "ssm:GetDocument",
                "ssm:DescribeDocument",
                "ssm:GetManifest",
                "ssm:GetParameter",
                "ssm:GetParameters",
                "ssm:ListAssociations",
                "ssm:ListInstanceAssociations",
                "ssm:PutInventory",
                "ssm:PutComplianceItems",
                "ssm:PutConfigurePackageResult",
                "ssm:UpdateAssociationStatus",
                "ssm:UpdateInstanceAssociationStatus",
                "ssm:UpdateInstanceInformation"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "ssmmessages:CreateControlChannel",
                "ssmmessages:CreateDataChannel",
                "ssmmessages:OpenControlChannel",
                "ssmmessages:OpenDataChannel"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "instance" {
  role       = aws_iam_role.instance.name
  policy_arn = aws_iam_policy.instance.arn
}

resource "aws_iam_role_policy_attachment" "cms_cloud_ssm_iam_policy_v3" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/cms-cloud-ssm-iam-policy-v3"
}

resource "aws_iam_role_policy_attachment" "amazon_elastic_file_system_full_access" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonElasticFileSystemFullAccess"
}

resource "aws_iam_role_policy_attachment" "amazon_s3_outposts_read_only_access" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3OutpostsReadOnlyAccess"
}

resource "aws_iam_role_policy_attachment" "amazon_s3_read_only_access" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"
}

resource "aws_iam_role_policy_attachment" "amazon_ssm_managed_instance_core" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "amazon_cloudwatch_agent_server_policy" {
  role       = aws_iam_role.instance.name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

/* ------- ACCESS LOG KMS ------- */
resource "aws_kms_key" "access_log_kms_key" {
  description             = local.access_log_kms_description
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "access_log_kms_alias" {
  name          = local.access_log_kms_alias
  target_key_id = aws_kms_key.access_log_kms_key.key_id
}

/* ------- APP CONFIG KMS ------- */
resource "aws_kms_key" "app_config_kms_key" {
  description             = local.app_config_kms_description
  deletion_window_in_days = 10
  enable_key_rotation     = true
  policy                  = <<EOF
  {
      "Version": "2012-10-17",
      "Id": "key-default-1",
      "Statement": [
            {
              "Sid": "Enable IAM User Permissions",
              "Effect": "Allow",
              "Principal": {
                  "AWS": "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
              },
              "Action": "kms:*",
              "Resource": "*"
          },
          {
              "Sid": "Enable IAM User Permissions",
              "Effect": "Allow",
              "Principal": {
                  "AWS": "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"
              },
              "Action": ["kms:GenerateDataKey*","kms:Decrypt"],
              "Resource": "*"
          }
      ]
  }
  EOF
}

resource "aws_kms_alias" "app_config_kms_alias" {
  name          = local.app_config_kms_alias
  target_key_id = aws_kms_key.app_config_kms_key.key_id
}

/* ------- EFS ------- */
resource "aws_kms_key" "efs_kms_key" {
  description             = local.efs_kms_description
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "efs_kms_alias" {
  name          = local.efs_kms_alias
  target_key_id = aws_kms_key.efs_kms_key.key_id
}

module "efs" {
  source = "../modules/efs"

  vpc_id  = local.vpc_id
  app     = local.app_name
  env     = var.env
  service = local.efs_service_name
  name    = local.efs_instance_name

  subnets = flatten([
    data.aws_subnets.private_subnets.ids,
  ])

  additional_ingress_sgs = flatten([
    aws_security_group.worker_sg.id,
    aws_security_group.app_sg.id,
  ])

  kms_key_id = "arn:aws:kms:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:key/${aws_kms_alias.efs_kms_alias.target_key_id}"
}

resource "aws_efs_access_point" "api" {
  file_system_id = module.efs.efs_id
  posix_user {
    uid = local.app_user_uid
    gid = local.app_user_gid
  }
}

resource "aws_efs_access_point" "worker" {
  file_system_id = module.efs.efs_id
  posix_user {
    uid = local.app_user_uid
    gid = local.app_user_gid
  }
}

/* ------- API and Worker Queue DB ------- */
resource "aws_kms_key" "rds_kms_key" {
  description             = local.rds_kms_description
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "rds_kms_alias" {
  name          = local.rds_kms_alias
  target_key_id = aws_kms_key.rds_kms_key.key_id
}

# Pare the standard GDIT security groups down
# to disallow connections to prod RDS instance
data "aws_security_groups" "rds_sgs" {
  filter {
    name = "tag:Name"

    values = local.rds_sg_filter_names
  }

  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
}

resource "aws_security_group" "sg_alb_01" {
  name        = local.alb_sg_name
  description = local.alb_sg_description
  vpc_id      = local.vpc_id

  tags = {
    "cms-cloud-exempt:open-sg" = local.sg_compliance_attestation_url
  }
}

resource "aws_security_group_rule" "alb_ingress_01" {
  #ts:skip=AC_AWS_0276 Allow traffic from any IP and let WAF filter
  type              = "ingress"
  from_port         = local.https_port
  to_port           = local.https_port
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  ipv6_cidr_blocks  = ["::/0"]
  security_group_id = aws_security_group.sg_alb_01.id
  description       = local.alb_ingress_description
}

resource "aws_security_group_rule" "alb_ingress_http_01" {
  #ts:skip=AC_AWS_0276 Allow traffic from any IP and let WAF filter
  type              = "ingress"
  from_port         = local.http_port
  to_port           = local.http_port
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  ipv6_cidr_blocks  = ["::/0"]
  security_group_id = aws_security_group.sg_alb_01.id
  description       = local.alb_http_ingress_description
}

resource "aws_security_group_rule" "alb_app_egress" {
  type                     = "egress"
  from_port                = local.app_port
  to_port                  = local.app_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.app_sg.id
  security_group_id        = aws_security_group.sg_alb_01.id
}

resource "aws_security_group_rule" "alb_worker_egress" {
  type                     = "egress"
  from_port                = local.app_port
  to_port                  = local.app_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.worker_sg.id
  security_group_id        = aws_security_group.sg_alb_01.id
}

/* ------ API ALB ------ */
resource "aws_lb" "alb_01" {
  name                       = local.alb_name
  internal                   = false
  load_balancer_type         = "application"
  enable_deletion_protection = true
  drop_invalid_header_fields = true

  subnets = data.aws_subnets.public_subnets.ids

  security_groups = [aws_security_group.sg_alb_01.id]

  access_logs {
    bucket  = "cms-cloud-${data.aws_caller_identity.current.account_id}-${data.aws_region.current.name}"
    enabled = true
  }

  enable_cross_zone_load_balancing = true
  idle_timeout                     = local.alb_idle_timeout

  tags = {
    Name = local.alb_tag_name
  }
}

# listeners
data "aws_acm_certificate" "web" {
  domain = local.web_domain
}

resource "aws_lb_listener" "alb_https_01" {
  load_balancer_arn = aws_lb.alb_01.arn
  port              = local.https_port
  protocol          = "HTTPS"
  ssl_policy        = var.ssl_policy
  certificate_arn   = data.aws_acm_certificate.web.arn

  default_action {
    type = "forward"

    forward {
      target_group {
        arn = aws_lb_target_group.ecs_api_https.arn
      }
    }
  }
}

/* -------- SSAS ALB SG-------- */
resource "aws_security_group" "ssas_alb" {
  name        = local.ssas_alb_sg_name
  description = local.ssas_alb_sg_description
  vpc_id      = local.vpc_id
}

resource "aws_security_group_rule" "ssas_alb_ingress_public" {
  type                     = "ingress"
  from_port                = local.https_port
  to_port                  = local.https_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.app_sg.id
  security_group_id        = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_admin" {
  type                     = "ingress"
  from_port                = local.admin_port
  to_port                  = local.admin_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.app_sg.id
  security_group_id        = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_aco_ms_admin" {
  type      = "ingress"
  from_port = local.admin_port
  to_port   = local.admin_port
  protocol  = "tcp"

  cidr_blocks = local.ssas_aco_ms_admin_cidr_blocks

  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_4i_public" {
  type      = "ingress"
  from_port = local.https_port
  to_port   = local.https_port
  protocol  = "tcp"

  cidr_blocks = local.ssas_4i_public_cidr_blocks

  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_4i_admin" {
  type      = "ingress"
  from_port = local.admin_port
  to_port   = local.admin_port
  protocol  = "tcp"

  cidr_blocks = local.ssas_4i_admin_cidr_blocks

  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_ihp_public" {
  type              = "ingress"
  from_port         = local.https_port
  to_port           = local.https_port
  protocol          = "tcp"
  cidr_blocks       = local.ssas_ihp_cidr_blocks
  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_ihp_admin" {
  type              = "ingress"
  from_port         = local.admin_port
  to_port           = local.admin_port
  protocol          = "tcp"
  cidr_blocks       = local.ssas_ihp_cidr_blocks
  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_gha_runners" {
  type      = "ingress"
  from_port = local.admin_port
  to_port   = local.admin_port
  protocol  = "tcp"

  cidr_blocks = local.ssas_gha_runners_cidr_blocks

  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_gha_runners_public" {
  type      = "ingress"
  from_port = local.https_port
  to_port   = local.https_port
  protocol  = "tcp"

  cidr_blocks = local.ssas_gha_runners_cidr_blocks

  security_group_id = aws_security_group.ssas_alb.id
}

data "aws_security_group" "remote_management" {
  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
  name = "remote-management"
}

resource "aws_security_group_rule" "ssas_alb_ingress_vpc" {
  type      = "ingress"
  from_port = local.admin_port
  to_port   = local.admin_port
  protocol  = "tcp"

  cidr_blocks = [
    local.app_cidr_block,
  ]

  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_vpc_public" {
  type      = "ingress"
  from_port = local.https_port
  to_port   = local.https_port
  protocol  = "tcp"

  cidr_blocks = [
    local.app_cidr_block,
  ]

  security_group_id = aws_security_group.ssas_alb.id
}

/* ------ SSAS ALB ------ */
resource "aws_lb" "ssas_alb" {
  name                       = local.ssas_alb_name
  internal                   = true
  load_balancer_type         = "application"
  enable_deletion_protection = true
  drop_invalid_header_fields = true

  subnets = data.aws_subnets.private_subnets.ids

  security_groups = flatten([
    data.aws_security_group.zscaler_private.id,
    aws_security_group.ssas_alb.id,
    data.aws_security_group.remote_management.id,
  ])

  access_logs {
    bucket  = "cms-cloud-${data.aws_caller_identity.current.account_id}-${data.aws_region.current.name}"
    enabled = true
  }

  enable_cross_zone_load_balancing = true
  idle_timeout                     = local.ssas_alb_idle_timeout

  tags = {
    Name = local.ssas_alb_tag_name
  }
}

# listeners
resource "aws_lb_listener" "ssas_alb_public" {
  load_balancer_arn = aws_lb.ssas_alb.arn
  port              = local.https_port
  protocol          = "HTTPS"
  ssl_policy        = var.ssl_policy
  certificate_arn   = local.ssas_certificate_arn

  default_action {
    type = "forward"

    forward {
      target_group {
        arn = aws_lb_target_group.ecs_ssas_public.arn
      }
    }
  }
}

resource "aws_lb_listener" "ssas_alb_admin" {
  load_balancer_arn = aws_lb.ssas_alb.arn
  port              = local.admin_port
  protocol          = "HTTPS"
  ssl_policy        = var.ssl_policy
  certificate_arn   = local.ssas_certificate_arn

  default_action {
    type = "forward"

    forward {
      target_group {
        arn = aws_lb_target_group.ecs_ssas_admin.arn
      }
    }
  }
}

/* ------ API/SSAS Security Group ------- */
resource "aws_security_group" "app_sg" {
  name        = local.app_sg_name
  description = local.app_sg_description
  vpc_id      = local.vpc_id

  tags = {
    Name = local.app_sg_name
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      id,
      description,
    ]
  }
}

resource "aws_security_group_rule" "app_sg_ingress_01" {
  type                     = "ingress"
  from_port                = local.app_port
  to_port                  = local.app_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.sg_alb_01.id
  security_group_id        = aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "ssas_sg_ingress_public" {
  type                     = "ingress"
  from_port                = local.ssas_public_port
  to_port                  = local.ssas_public_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.ssas_alb.id
  security_group_id        = aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "ssas_sg_ingress_admin" {
  type                     = "ingress"
  from_port                = local.ssas_admin_port
  to_port                  = local.ssas_admin_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.ssas_alb.id
  security_group_id        = aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "peering_app_sg_ingress" {
  type              = "ingress"
  from_port         = local.ssh_port
  to_port           = local.ssh_port
  protocol          = "tcp"
  security_group_id = aws_security_group.app_sg.id
  cidr_blocks       = [local.management_cidr_block]
}

resource "aws_security_group_rule" "app_sg_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = [local.all_zero_cidr]
  security_group_id = aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "db_ingress_from_api" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = local.postgres_port
  protocol                 = "tcp"
  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = aws_security_group.app_sg.id
  description              = "API SG access"
}

/* ---- Worker Security Group ----- */
resource "aws_security_group" "worker_sg" {
  name        = local.worker_sg_name
  description = local.worker_sg_description
  vpc_id      = local.vpc_id

  tags = {
    Name = local.worker_sg_name
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      id,
      description,
    ]
  }
}

resource "aws_security_group_rule" "worker_sg_ingress_01" {
  type                     = "ingress"
  from_port                = local.app_port
  to_port                  = local.app_port
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.sg_alb_01.id
  security_group_id        = aws_security_group.worker_sg.id
}

resource "aws_security_group_rule" "peering_worker_sg_ingress" {
  type              = "ingress"
  from_port         = local.ssh_port
  to_port           = local.ssh_port
  protocol          = "tcp"
  security_group_id = aws_security_group.worker_sg.id
  cidr_blocks       = [local.management_cidr_block]
}

resource "aws_security_group_rule" "worker_sg_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = [local.all_zero_cidr]
  security_group_id = aws_security_group.worker_sg.id
}

resource "aws_security_group_rule" "db_ingress_from_worker" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = local.postgres_port
  protocol                 = "tcp"
  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = aws_security_group.worker_sg.id
  description              = "Worker SG access"
}

/* ---- CloudWatch ----- */
resource "aws_sns_topic" "cloudwatch_alarms_topic" {
  display_name      = local.cloudwatch_alarms_topic_name
  name              = local.cloudwatch_alarms_topic_name
  kms_master_key_id = "alias/${module.platform.app}-${module.platform.env}"
}

# CDAP-managed alarm-to-slack service queue
data "aws_sqs_queue" "alarm_to_slack" {
  name = "bcda-prod-alarm-to-slack"
}

resource "aws_sns_topic_subscription" "alarm_to_slack" {
  topic_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  protocol  = "sqs"
  endpoint  = data.aws_sqs_queue.alarm_to_slack.arn
}

resource "aws_sns_topic" "cloudwatch_critical_alarms_topic" {
  name              = local.cloudwatch_critical_alarms_topic_name
  kms_master_key_id = "alias/aws/sns"
}

module "bcda_alarms" {
  source = "../modules/bcda_alarms"

  env                         = var.env
  cloudwatch_notification_arn = aws_sns_topic.cloudwatch_critical_alarms_topic.arn
}

module "cloudwatch_alarms_elb_http" {
  source = "../modules/elb_http_alarms"

  app                         = local.app_name
  env                         = var.env
  vpc_name                    = data.aws_vpc.main.id
  cloudwatch_notification_arn = aws_sns_topic.cloudwatch_critical_alarms_topic.arn
  load_balancer_name          = aws_lb.alb_01.name

  alarm_backend_4xx_enable = true
  alarm_backend_5xx_enable = true
  alarm_elb_5xx_enable     = true

  alarm_elb_no_backend_treat_missing_data   = local.alarm_elb_no_backend_treat_missing_data
  alarm_elb_high_latency_treat_missing_data = local.alarm_elb_high_latency_treat_missing_data
}

data "aws_rds_cluster" "clusterName" {
  cluster_identifier = "bcda-${var.env}-aurora"
}

module "cloudwatch_alarms_efs" {
  source = "../modules/efs_alarms"

  app                         = local.app_name
  env                         = var.env
  cloudwatch_notification_arn = aws_sns_topic.cloudwatch_alarms_topic.arn

  efs_name = module.efs.efs_id
}

data "aws_route_table" "main" {
  count = length(distinct(sort(data.aws_subnets.private_subnets.ids)))
  subnet_id = element(
    distinct(sort(data.aws_subnets.private_subnets.ids)),
    count.index,
  )
}

module "aco_creds_bucket" {
  source = "../modules/s3"

  env  = var.env
  name = "bcda-${var.env}-aco-creds"
}

resource "aws_ssm_parameter" "aco_creds_bucket" {
  name      = "/bcda/${var.env}/sensitive/aco_creds_bucket"
  type      = "String"
  value     = module.aco_creds_bucket.id
  overwrite = true
}

module "config_bucket" {
  source = "../modules/s3"
  env    = var.env
  name   = "bcda-${var.env}-config"
}

resource "aws_ssm_parameter" "config_bucket_params" {
  for_each  = toset(["api", "ssas", "worker"])
  name      = "/bcda/${var.env}/sensitive/${each.value}/CONFIG_BUCKET"
  type      = "String"
  value     = module.config_bucket.id
  overwrite = true
}

resource "aws_s3_bucket_lifecycle_configuration" "clean_up_objects" {
  bucket = module.aco_creds_bucket.id

  rule {
    id     = "Clean-up-Objects"
    status = "Enabled"

    filter {}

    expiration {
      days = 30
    }
  }
}

/* ---- Insights ----- */

module "insights" {
  source                = "../insights"
  env                   = var.env
  worker_security_group = aws_security_group.worker_sg.id
  db_subnet_group       = "bcda-${var.env}-rds-subnets"
}

module "get_job_data" {
  source            = "../modules/insights_data_sampler"
  name              = "get_job_data"
  description       = "gets data related to job requests over the last 30 days"
  schedule          = "rate(10 minutes)"
  query             = file("${path.module}/../insights/queries/get_job_data.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_stale_pending_jobs" {
  source            = "../modules/insights_data_sampler"
  name              = "get_stale_pending_jobs"
  description       = "get jobs with 'Pending' or 'In Progress' status that are older than 4 hours"
  schedule          = "rate(10 minutes)"
  query             = file("${path.module}/../insights/queries/get_stale_pending_jobs.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_active_acos" {
  source            = "../modules/insights_data_sampler"
  name              = "get_active_acos"
  description       = "gets data about the active (credentialed) ACOs"
  schedule          = "rate(10 minutes)"
  query             = file("${path.module}/../insights/queries/get_active_acos.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_stale_cclf_imports" {
  source            = "../modules/insights_data_sampler"
  name              = "get_stale_cclf_imports"
  description       = "get ACOs whose last successful cclf import was more than 40 days ago"
  schedule          = "rate(1 day)"
  query             = file("${path.module}/../insights/queries/get_stale_cclf_imports.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_num_benes_per_aco" {
  source            = "../modules/insights_data_sampler"
  name              = "get_num_benes_per_aco"
  description       = "gets the total beneficiary attribution per ACO based on latest CCLF file"
  schedule          = "rate(1 day)"
  query             = file("${path.module}/../insights/queries/get_num_benes_per_aco.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_suppression_metrics" {
  source            = "../modules/insights_data_sampler"
  name              = "get_suppression_metrics"
  description       = "gets the number of suppressed beneficiaries per opt-out file over the last 90 days"
  schedule          = "rate(1 day)"
  query             = file("${path.module}/../insights/queries/get_suppression_metrics.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_num_days_to_make_first_request" {
  source            = "../modules/insights_data_sampler"
  name              = "get_num_days_to_make_first_request"
  description       = "gets the number of days it took all ACOs (active or inactive) to make their first data request after onboarding (since 2019-09-11) "
  schedule          = "rate(1 day)"
  query             = file("${path.module}/../insights/queries/get_num_days_to_make_first_request.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

module "get_acos_with_expired_credentials" {
  source            = "../modules/insights_data_sampler"
  name              = "get_acos_with_expired_credentials"
  description       = "gets a list of ACOs who have onboarded but have let their API keys expire"
  schedule          = "rate(1 day)"
  query             = file("${path.module}/../insights/queries/get_expired_credentials.sql")
  env               = "prod"
  insights_role_arn = module.insights.insights_role_arn
  lambda_arn        = module.insights.lambda_arn
}

data "aws_ssm_parameter" "db_admin_password" {
  name = "/bcda/${var.env}/sensitive/db_admin_password"
}

module "platform" {
  source    = "github.com/CMSgov/cdap.git//terraform/modules/platform?ref=ff2ef539fb06f2c98f0e3ce0c8f922bdacb96d66"
  providers = { aws = aws, aws.secondary = aws.secondary }

  app         = "bcda"
  env         = var.env
  root_module = "github.com/CMSgov/bcda-app/tree/main/terraform/${var.env}"
  service     = "bcda"
}

data "aws_iam_policy_document" "kms" {
  statement {
    sid = "AllowEnvCMKAccess"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
      "kms:ReEncryptFrom",
      "kms:ReEncryptTo",
      "kms:DescribeKey",
      "kms:CreateGrant",
      "kms:ListGrants",
      "kms:RevokeGrant"
    ]
    resources = [module.platform.kms_alias_primary.target_key_arn]
  }
}

resource "aws_iam_policy" "kms" {
  name   = "${module.platform.app}-${module.platform.env}-kms"
  path   = "/delegatedadmin/developer/"
  policy = data.aws_iam_policy_document.kms.json
}

data "aws_iam_policy_document" "kms_key_access" {
  statement {
    sid = "AllowEnvCMKAccess"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
      "kms:ReEncrypt",
      "kms:DescribeKey",
      "kms:Encrypt"
    ]
    resources = [module.platform.kms_alias_primary.target_key_arn]
  }
}

data "aws_iam_policy" "developer_boundary_policy" {
  name = "developer-boundary-policy"
}

resource "aws_iam_policy" "kms_key_access" {
  name        = "${module.platform.app}-${module.platform.env}-kms-key-access"
  path        = "/delegatedadmin/developer/"
  description = "Permissions to access environment ${module.platform.env} KMS CMK"
  policy      = data.aws_iam_policy_document.kms_key_access.json
}

data "aws_iam_policy" "rds_monitoring" {
  name = "AmazonRDSEnhancedMonitoringRole"
}

data "aws_iam_policy_document" "rds_monitoring_assume" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["monitoring.rds.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "db_monitoring" {
  name                 = "${module.platform.app}-${module.platform.env}-rds-monitoring"
  assume_role_policy   = data.aws_iam_policy_document.rds_monitoring_assume.json
  path                 = "/delegatedadmin/developer/"
  permissions_boundary = data.aws_iam_policy.developer_boundary_policy.arn
}

resource "aws_iam_role_policy_attachment" "db_monitoring" {
  role       = aws_iam_role.db_monitoring.name
  policy_arn = data.aws_iam_policy.rds_monitoring.arn
}

resource "aws_iam_role_policy_attachment" "db_monitoring_kms" {
  role       = aws_iam_role.db_monitoring.name
  policy_arn = aws_iam_policy.kms_key_access.arn
}

module "db" {
  source = "github.com/CMSgov/cdap/terraform/modules/aurora?ref=6c23dd96089b69c34cd3ca8766789e18535b89ad"

  backup_window         = "04:17-04:47"
  cluster_identifier    = "bcda-${var.env}-aurora"
  deletion_protection   = !module.platform.is_ephemeral_env
  instance_class        = "db.r8g.large"
  instance_count        = 2
  maintenance_window    = "sun:23:08-sun:23:38"
  monitoring_interval   = var.monitoring_interval
  monitoring_role_arn   = aws_iam_role.db_monitoring.arn
  username              = "bcda"
  password              = data.aws_ssm_parameter.db_admin_password.value
  platform              = module.platform
  subnet_group_override = "bcda-${var.env}-rds-subnets"

  cluster_instance_parameters = [
    {
      apply_method = "immediate"
      name         = "random_page_cost"
      value        = "1.1"
    },
    {
      apply_method = "immediate"
      name         = "work_mem"
      value        = "32768"
    }
  ]
}

# Fargate--------------------------------------------------

module "ecs_cluster" {
  source                = "github.com/CMSgov/cdap/terraform/modules/cluster?ref=86e705b7a0d81ee1f481948678092ed47ba32741"
  cluster_name_override = "bcda-${var.env}"
  platform              = module.platform
}

# API

# this resource is shared across envs.  see sandbox/main.tf for resource.
data "aws_ecr_repository" "ecr_api" {
  name = "bcda-api"
}

resource "aws_lb_target_group" "ecs_api_https" {
  name                 = "bcda-${var.env}-ecs-api-https"
  port                 = local.app_port
  protocol             = "HTTPS"
  vpc_id               = local.vpc_id
  deregistration_delay = local.alb_deregistration_delay
  target_type          = "ip"

  health_check {
    interval            = local.alb_health_check_interval
    path                = local.alb_health_check_path
    port                = local.app_port
    protocol            = "HTTPS"
    healthy_threshold   = local.alb_health_check_healthy_threshold
    unhealthy_threshold = local.alb_health_check_unhealthy_threshold
    timeout             = local.alb_health_check_timeout
  }
}

data "aws_ssm_parameter" "params_api" {
  for_each        = toset(local.param_names_api)
  name            = "/bcda/${var.env}/sensitive/api/${each.value}"
  with_decryption = true
}

module "ecs_api" {
  source                = "github.com/CMSgov/cdap/terraform/modules/service?ref=dde50eb21f7b6ecc53ddb2483ba66d867e39c604"
  service_name_override = "api"
  platform              = module.platform
  cluster_arn           = module.ecs_cluster.this.arn
  image                 = "${data.aws_ecr_repository.ecr_api.repository_url}:${var.api_image_tag}"
  cpu                   = local.ecs_task_def_cpu_api
  memory                = local.ecs_task_def_mem_api
  desired_count         = local.api_desired_min
  port_mappings         = [{ containerPort = local.app_port }]
  security_groups       = [aws_security_group.app_sg.id]
  task_role_arn         = aws_iam_role.instance.arn
  cpu_architecture      = "ARM64"

  container_environment = [
    { name = "LOG_TO_STD_OUT", value = "true" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_api : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  load_balancers = [{
    target_group_arn = aws_lb_target_group.ecs_api_https.arn
    container_name   = local.container_name_api
    container_port   = local.app_port
  }]

  mount_points = [
    {
      containerPath = "/var/efs",
      sourceVolume  = "efs"
    },
    {
      containerPath = "/etc/sv/api/env"
      sourceVolume  = "api_config"
    }
  ]

  volumes = [
    {
      name = "efs"
      efs_volume_configuration = {
        file_system_id     = module.efs.efs_id
        root_directory     = "/"
        transit_encryption = "ENABLED"
        authorization_config = {
          access_point_id = aws_efs_access_point.api.id
        }
      }
    },
    { name = "api_config" }
  ]
}

module "api-ecs-alarms" {
  source = "../modules/ecs_alarms"

  service_name = module.ecs_api.service.name
  cluster_name = module.ecs_cluster.this.name

  alarm_notification_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = aws_sns_topic.cloudwatch_alarms_topic.arn

  alarms = [
    {
      alarm_name        = "${module.ecs_api.service.name}-cpu-critical"
      alarm_description = "CRITICAL - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-cpu-warn"
      alarm_description = "WARN - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-memory-critical"
      alarm_description = "CRITICAL - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_api.service.name}-memory-warn"
      alarm_description = "WARN - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    }
  ]
}

# SSAS

# this resource is shared across envs.  see sandbox/main.tf for resource.
data "aws_ecr_repository" "ecr_ssas" {
  name = "bcda-ssas"
}

resource "aws_lb_target_group" "ecs_ssas_public" {
  name                 = "bcda-${var.env}-ecs-ssas-public"
  port                 = local.ssas_public_port
  protocol             = "HTTPS"
  vpc_id               = local.vpc_id
  deregistration_delay = local.alb_deregistration_delay
  target_type          = "ip"

  health_check {
    interval            = local.alb_health_check_interval
    path                = local.alb_health_check_path
    port                = local.ssas_public_port
    protocol            = "HTTPS"
    healthy_threshold   = local.alb_health_check_healthy_threshold
    unhealthy_threshold = local.alb_health_check_unhealthy_threshold
    timeout             = local.alb_health_check_timeout
  }
}

resource "aws_lb_target_group" "ecs_ssas_admin" {
  name                 = "bcda-${var.env}-ecs-ssas-admin"
  port                 = local.ssas_admin_port
  protocol             = "HTTPS"
  vpc_id               = local.vpc_id
  deregistration_delay = local.alb_deregistration_delay
  target_type          = "ip"

  health_check {
    interval            = local.alb_health_check_interval
    path                = local.alb_health_check_path
    port                = local.ssas_admin_port
    protocol            = "HTTPS"
    healthy_threshold   = local.alb_health_check_healthy_threshold
    unhealthy_threshold = local.alb_health_check_unhealthy_threshold
    timeout             = local.alb_health_check_timeout
  }
}

data "aws_ssm_parameter" "params_ssas" {
  for_each        = toset(local.param_names_ssas)
  name            = "/bcda/${var.env}/sensitive/ssas/${each.value}"
  with_decryption = true
}

module "ecs_ssas" {
  source                = "github.com/CMSgov/cdap/terraform/modules/service?ref=dde50eb21f7b6ecc53ddb2483ba66d867e39c604"
  service_name_override = "ssas"
  platform              = module.platform
  cluster_arn           = module.ecs_cluster.this.arn
  image                 = "${data.aws_ecr_repository.ecr_ssas.repository_url}:${var.ssas_image_tag}"
  cpu                   = local.ecs_task_def_cpu_ssas
  memory                = local.ecs_task_def_mem_ssas
  desired_count         = local.ssas_desired_min
  port_mappings         = [{ containerPort = local.ssas_public_port }, { containerPort = local.ssas_admin_port }]
  security_groups       = [aws_security_group.app_sg.id]
  task_role_arn         = aws_iam_role.instance.arn
  cpu_architecture      = "ARM64"

  container_environment = [
    { name = "LOG_TO_STD_OUT", value = "true" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_ssas : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  load_balancers = [
    {
      target_group_arn = aws_lb_target_group.ecs_ssas_public.arn
      container_name   = local.container_name_ssas
      container_port   = local.ssas_public_port
    },
    {
      target_group_arn = aws_lb_target_group.ecs_ssas_admin.arn
      container_name   = local.container_name_ssas
      container_port   = local.ssas_admin_port
    }
  ]

  mount_points = [
    {
      containerPath = "/etc/sv/ssas/env"
      sourceVolume  = "ssas_config"
    }
  ]

  volumes = [{ name = "ssas_config" }]
}

module "ssas-ecs-alarms" {
  source = "../modules/ecs_alarms"

  service_name = module.ecs_ssas.service.name
  cluster_name = module.ecs_cluster.this.name

  alarm_notification_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = aws_sns_topic.cloudwatch_alarms_topic.arn

  alarms = [
    {
      alarm_name        = "${module.ecs_ssas.service.name}-cpu-critical"
      alarm_description = "CRITICAL - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-cpu-warn"
      alarm_description = "WARN - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-memory-critical"
      alarm_description = "CRITICAL - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_ssas.service.name}-memory-warn"
      alarm_description = "WARN - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    }
  ]
}

# WORKER

# this resource is shared across envs.  see sandbox/main.tf for resource.
data "aws_ecr_repository" "ecr_worker" {
  name = "bcda-worker"
}

data "aws_ssm_parameter" "params_worker" {
  for_each        = toset(local.param_names_worker)
  name            = "/bcda/${var.env}/sensitive/worker/${each.value}"
  with_decryption = true
}

module "ecs_worker" {
  source                = "github.com/CMSgov/cdap/terraform/modules/service?ref=dde50eb21f7b6ecc53ddb2483ba66d867e39c604"
  service_name_override = "worker"
  platform              = module.platform
  cluster_arn           = module.ecs_cluster.this.arn
  image                 = "${data.aws_ecr_repository.ecr_worker.repository_url}:${var.worker_image_tag}"
  cpu                   = local.ecs_task_def_cpu_worker
  memory                = local.ecs_task_def_mem_worker
  desired_count         = local.worker_desired_min
  security_groups       = [aws_security_group.worker_sg.id]
  task_role_arn         = aws_iam_role.instance.arn
  cpu_architecture      = "ARM64"

  container_environment = [
    { name = "FHIR_TEMP_DIR", value = "/home/bcda/FHIR_TEMP_DIR" }
  ]

  container_secrets = [
    for param in data.aws_ssm_parameter.params_worker : {
      name      = element(split("/", param.name), -1)
      valueFrom = param.arn
    }
  ]

  health_check = {
    command     = ["CMD-SHELL", "bcdaworker health >> /proc/1/fd/1 2>&1  || exit 1"],
    interval    = local.alb_health_check_interval,
    retries     = local.alb_health_check_unhealthy_threshold,
    startPeriod = local.health_check_start_period,
    timeout     = local.alb_health_check_timeout
  }

  mount_points = [
    {
      containerPath = "/var/efs",
      sourceVolume  = "efs"
    },
    {
      containerPath = "/etc/sv/worker/env"
      sourceVolume  = "worker_config"
    },
    {
      containerPath = "/home/bcda/FHIR_TEMP_DIR",
      sourceVolume  = "fhir_tmp"
    }
  ]

  volumes = [
    {
      name = "efs"
      efs_volume_configuration = {
        file_system_id     = module.efs.efs_id
        root_directory     = "/"
        transit_encryption = "ENABLED"
        authorization_config = {
          access_point_id = aws_efs_access_point.worker.id
        }
      }
    },
    { name = "worker_config" },
    { name = "fhir_tmp" }
  ]
}

module "worker-ecs-alarms" {
  source = "../modules/ecs_alarms"

  service_name = module.ecs_worker.service.name
  cluster_name = module.ecs_cluster.this.name

  alarm_notification_arn = aws_sns_topic.cloudwatch_alarms_topic.arn
  ok_notification_arn    = aws_sns_topic.cloudwatch_alarms_topic.arn

  alarms = [
    {
      alarm_name        = "${module.ecs_worker.service.name}-cpu-critical"
      alarm_description = "CRITICAL - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_worker.service.name}-cpu-warn"
      alarm_description = "WARN - CPU is too high for service"
      metric_name       = "CPUUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    },
    {
      alarm_name        = "${module.ecs_worker.service.name}-memory-critical"
      alarm_description = "CRITICAL - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 90
    },
    {
      alarm_name        = "${module.ecs_worker.service.name}-memory-warn"
      alarm_description = "WARN - memory is too high for service"
      metric_name       = "MemoryUtilization"
      period            = 60,
      eval_periods      = 5,
      threshold         = 75
    }
  ]
}

# ECS scaling: targets, policies, and alarms

resource "aws_appautoscaling_target" "ecs_api_cpu_target" {
  min_capacity       = local.api_desired_min
  max_capacity       = local.api_desired_max
  resource_id        = "service/${module.ecs_cluster.this.name}/${module.ecs_api.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_api_cpu_policy" {
  name               = "bcda-${var.env}-api-cpu-auto-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_api_cpu_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_api_cpu_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_api_cpu_target.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 30
    scale_in_cooldown  = 180
    scale_out_cooldown = 180

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

resource "aws_appautoscaling_target" "ecs_ssas_cpu_target" {
  min_capacity       = local.ssas_desired_min
  max_capacity       = local.ssas_desired_max
  resource_id        = "service/${module.ecs_cluster.this.name}/${module.ecs_ssas.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_ssas_cpu_policy" {
  name               = "bcda-${var.env}-ssas-cpu-auto-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_ssas_cpu_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_ssas_cpu_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_ssas_cpu_target.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 20
    scale_in_cooldown  = 720
    scale_out_cooldown = 60

    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
  }
}

resource "aws_appautoscaling_target" "ecs_worker_target" {
  min_capacity       = local.worker_desired_min
  max_capacity       = local.worker_desired_max
  resource_id        = "service/${module.ecs_cluster.this.name}/${module.ecs_worker.service.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_worker_policy" {
  name               = "bcda-${var.env}-worker-job-count-scaleup"
  policy_type        = "StepScaling"
  resource_id        = aws_appautoscaling_target.ecs_worker_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_worker_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_worker_target.service_namespace

  step_scaling_policy_configuration {
    adjustment_type         = "ExactCapacity"
    cooldown                = 300 # wait 5m before attempting to scale again
    metric_aggregation_type = "Maximum"

    step_adjustment {
      scaling_adjustment          = local.worker_desired_min
      metric_interval_upper_bound = 25
    }

    step_adjustment {
      scaling_adjustment          = local.worker_desired_min + 2
      metric_interval_lower_bound = 25
      metric_interval_upper_bound = 250
    }

    step_adjustment {
      scaling_adjustment          = local.worker_desired_min + 6
      metric_interval_lower_bound = 250
      metric_interval_upper_bound = 1000
    }

    step_adjustment {
      scaling_adjustment          = local.worker_desired_min + 8
      metric_interval_lower_bound = 1000
      metric_interval_upper_bound = 2500
    }

    step_adjustment {
      scaling_adjustment          = local.worker_desired_min + 10
      metric_interval_lower_bound = 2500
      metric_interval_upper_bound = 5000
    }

    step_adjustment {
      scaling_adjustment          = local.worker_desired_max
      metric_interval_lower_bound = 5000
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "ecs_worker_job_count" {
  alarm_name          = "bcda-${var.env}-worker-job-count"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = "2"
  metric_name         = "JobQueueCount"
  namespace           = "BCDA"
  period              = "120"
  statistic           = "Average"
  threshold           = "0"

  dimensions = {
    Environment = var.env
  }

  alarm_description = "Queued job count for workers"
  alarm_actions     = [aws_appautoscaling_policy.ecs_worker_policy.arn]
}

resource "aws_cloudwatch_log_group" "ecs_containerinsights" {
  name              = "/aws/ecs/containerinsights/bcda-${var.env}/performance"
  retention_in_days = var.log_retention_in_days

  tags = module.platform.default_tags
}

resource "aws_cloudwatch_log_group" "ecs_api" {
  name              = "/aws/ecs/fargate/bcda-${var.env}/api"
  retention_in_days = var.log_retention_in_days

  tags = module.platform.default_tags
}

resource "aws_cloudwatch_log_group" "ecs_ssas" {
  name              = "/aws/ecs/fargate/bcda-${var.env}/ssas"
  retention_in_days = var.log_retention_in_days

  tags = module.platform.default_tags
}

resource "aws_cloudwatch_log_group" "ecs_worker" {
  name              = "/aws/ecs/fargate/bcda-${var.env}/worker"
  retention_in_days = var.log_retention_in_days

  tags = module.platform.default_tags
}
