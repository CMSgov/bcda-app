resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_admin" {
  security_group_id            = data.aws_security_group.ssas_alb.id
  referenced_security_group_id = data.aws_security_group.app_sg.id
  from_port                    = local.config.ports.admin_port
  to_port                      = local.config.ports.admin_port
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_public" {
  security_group_id            = data.aws_security_group.ssas_alb.id
  referenced_security_group_id = data.aws_security_group.app_sg.id
  from_port                    = local.config.ports.https_port
  to_port                      = local.config.ports.https_port
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_vpc" {
  security_group_id = data.aws_security_group.ssas_alb.id
  cidr_ipv4         = local.app_cidr_block
  from_port         = local.config.ports.admin_port
  to_port           = local.config.ports.admin_port
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_vpc_public" {
  security_group_id = data.aws_security_group.ssas_alb.id
  cidr_ipv4         = local.app_cidr_block
  from_port         = local.config.ports.https_port
  to_port           = local.config.ports.https_port
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_gha_runners" {
  security_group_id = data.aws_security_group.ssas_alb.id
  cidr_ipv4         = local.management_cidr_block
  from_port         = local.config.ports.admin_port
  to_port           = local.config.ports.admin_port
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_alb_ingress_gha_runners_public" {
  security_group_id = data.aws_security_group.ssas_alb.id
  cidr_ipv4         = local.management_cidr_block
  from_port         = local.config.ports.https_port
  to_port           = local.config.ports.https_port
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "ssas_alb_egress_app" {
  security_group_id            = data.aws_security_group.ssas_alb.id
  referenced_security_group_id = data.aws_security_group.app_sg.id
  ip_protocol                  = "-1"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_sg_ingress_admin" {
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.ssas_alb.id
  from_port                    = local.config.ports.ssas_admin_port
  to_port                      = local.config.ports.ssas_admin_port
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_sg_ingress_public" {
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.ssas_alb.id
  from_port                    = local.config.ports.ssas_public_port
  to_port                      = local.config.ports.ssas_public_port
  ip_protocol                  = "tcp"
}
