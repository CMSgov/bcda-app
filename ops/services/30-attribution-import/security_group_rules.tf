data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_vpc_security_group_egress_rule" "egress_to_db" {
  from_port   = 5432
  to_port     = 5432
  ip_protocol = "tcp"
  description = "Allow DB access"

  security_group_id            = module.attribution_import_function.security_group_id
  referenced_security_group_id = data.aws_security_group.db.id
}

resource "aws_security_group_rule" "function_access" {
  type        = "ingress"
  from_port   = 5432
  to_port     = 5432
  protocol    = "tcp"
  description = "${local.full_name} function access"

  security_group_id        = data.aws_security_group.db.id
  source_security_group_id = module.attribution_import_function.security_group_id
}

