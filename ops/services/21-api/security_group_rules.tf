resource "aws_vpc_security_group_ingress_rule" "app_sg_ingress_01" {
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.sg_alb_01.id
  from_port                    = local.config.ports.app
  to_port                      = local.config.ports.app
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "app_sg_ingress_http_01" {
  count                        = contains(["prod"], local.parent_env) ? 0 : 1
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.sg_alb_01.id
  from_port                    = local.config.ports.app_http
  to_port                      = local.config.ports.app_http
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_sg_ingress_public" {
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.ssas_alb.id
  from_port                    = local.config.ports.ssas_public
  to_port                      = local.config.ports.ssas_public
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "ssas_sg_ingress_admin" {
  security_group_id            = data.aws_security_group.app_sg.id
  referenced_security_group_id = data.aws_security_group.ssas_alb.id
  from_port                    = local.config.ports.ssas_admin
  to_port                      = local.config.ports.ssas_admin
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "peering_app_sg_ingress" {
  security_group_id = data.aws_security_group.app_sg.id
  cidr_ipv4         = module.platform.platform_cidr
  from_port         = local.config.ports.ssh
  to_port           = local.config.ports.ssh
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "app_sg_egress" {
  security_group_id = data.aws_security_group.app_sg.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

resource "aws_vpc_security_group_ingress_rule" "alb_ingress_01" {
  security_group_id = data.aws_security_group.sg_alb_01.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = local.config.ports.https
  to_port           = local.config.ports.https
  ip_protocol       = "tcp"
  description       = "Ingress"
}

resource "aws_vpc_security_group_ingress_rule" "alb_ingress_01_ipv6" {
  security_group_id = data.aws_security_group.sg_alb_01.id
  cidr_ipv6         = "::/0"
  from_port         = local.config.ports.https
  to_port           = local.config.ports.https
  ip_protocol       = "tcp"
  description       = "Ingress"
}

resource "aws_vpc_security_group_ingress_rule" "alb_ingress_http_01" {
  security_group_id = data.aws_security_group.sg_alb_01.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = local.config.ports.http
  to_port           = local.config.ports.http
  ip_protocol       = "tcp"
  description       = "Ingress HTTP"
}

resource "aws_vpc_security_group_ingress_rule" "alb_ingress_http_01_ipv6" {
  security_group_id = data.aws_security_group.sg_alb_01.id
  cidr_ipv6         = "::/0"
  from_port         = local.config.ports.http
  to_port           = local.config.ports.http
  ip_protocol       = "tcp"
  description       = "Ingress HTTP"
}

resource "aws_vpc_security_group_egress_rule" "alb_app_egress" {
  security_group_id            = data.aws_security_group.sg_alb_01.id
  referenced_security_group_id = data.aws_security_group.app_sg.id
  from_port                    = local.config.ports.app
  to_port                      = local.config.ports.app
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "alb_worker_egress" {
  security_group_id            = data.aws_security_group.sg_alb_01.id
  referenced_security_group_id = data.aws_security_group.worker_sg.id
  from_port                    = local.config.ports.app
  to_port                      = local.config.ports.app
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "db_ingress_from_api" {
  security_group_id            = data.aws_security_group.db.id
  referenced_security_group_id = data.aws_security_group.app_sg.id
  from_port                    = local.config.ports.postgres
  to_port                      = local.config.ports.postgres
  ip_protocol                  = "tcp"
  description                  = "API SG access"
}
