resource "aws_security_group_rule" "ssas_alb_ingress_admin" {
  type                     = "ingress"
  from_port                = local.config.ports.admin_port
  to_port                  = local.config.ports.admin_port
  protocol                 = "tcp"
  security_group_id        = aws_security_group.ssas_alb.id
  source_security_group_id = data.aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_public" {
  type                     = "ingress"
  from_port                = local.config.ports.https_port
  to_port                  = local.config.ports.https_port
  protocol                 = "tcp"
  security_group_id        = aws_security_group.ssas_alb.id
  source_security_group_id = data.aws_security_group.app_sg.id
}

resource "aws_security_group_rule" "ssas_alb_ingress_vpc" {
  type              = "ingress"
  from_port         = local.config.ports.admin_port
  to_port           = local.config.ports.admin_port
  protocol          = "tcp"
  security_group_id = aws_security_group.ssas_alb.id
  cidr_blocks       = [local.app_cidr_block]
}

resource "aws_security_group_rule" "ssas_alb_ingress_vpc_public" {
  type              = "ingress"
  from_port         = local.config.ports.https_port
  to_port           = local.config.ports.https_port
  protocol          = "tcp"
  security_group_id = aws_security_group.ssas_alb.id
  cidr_blocks       = [local.app_cidr_block]
}

resource "aws_security_group_rule" "ssas_alb_ingress_gha_runners" {
  type              = "ingress"
  from_port         = local.config.ports.admin_port
  to_port           = local.config.ports.admin_port
  protocol          = "tcp"
  security_group_id = aws_security_group.ssas_alb.id
  cidr_blocks       = [local.management_cidr_block]
}

resource "aws_security_group_rule" "ssas_alb_ingress_gha_runners_public" {
  type              = "ingress"
  from_port         = local.config.ports.https_port
  to_port           = local.config.ports.https_port
  protocol          = "tcp"
  security_group_id = aws_security_group.ssas_alb.id
  cidr_blocks       = [local.management_cidr_block]
}

resource "aws_security_group_rule" "ssas_allow_all_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_sg_ingress_admin" {
  type                     = "ingress"
  from_port                = local.config.ports.ssas_admin_port
  to_port                  = local.config.ports.ssas_admin_port
  protocol                 = "tcp"
  security_group_id        = data.aws_security_group.app_sg.id
  source_security_group_id = aws_security_group.ssas_alb.id
}

resource "aws_security_group_rule" "ssas_sg_ingress_public" {
  type                     = "ingress"
  from_port                = local.config.ports.ssas_public_port
  to_port                  = local.config.ports.ssas_public_port
  protocol                 = "tcp"
  security_group_id        = data.aws_security_group.app_sg.id
  source_security_group_id = aws_security_group.ssas_alb.id
}
