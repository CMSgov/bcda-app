resource "aws_vpc_security_group_ingress_rule" "worker_sg_ingress_01" {
  security_group_id            = data.aws_security_group.worker_sg.id
  referenced_security_group_id = data.aws_security_group.sg_alb_01.id
  from_port                    = local.config.ports.app
  to_port                      = local.config.ports.app
  ip_protocol                  = "tcp"
}

resource "aws_vpc_security_group_ingress_rule" "peering_worker_sg_ingress" {
  security_group_id = data.aws_security_group.worker_sg.id
  cidr_ipv4         = module.platform.platform_cidr
  from_port         = local.config.ports.ssh
  to_port           = local.config.ports.ssh
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "worker_sg_egress" {
  security_group_id = data.aws_security_group.worker_sg.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

resource "aws_vpc_security_group_ingress_rule" "db_ingress_from_worker" {
  security_group_id            = data.aws_security_group.db.id
  referenced_security_group_id = data.aws_security_group.worker_sg.id
  from_port                    = local.config.ports.postgres
  to_port                      = local.config.ports.postgres
  ip_protocol                  = "tcp"
  description                  = "Worker SG access"
}
