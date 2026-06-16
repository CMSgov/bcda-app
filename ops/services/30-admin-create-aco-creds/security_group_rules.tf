data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_vpc_security_group_ingress_rule" "function_db_access" {
  from_port   = 5432
  to_port     = 5432
  ip_protocol = "tcp"
  description = "admin-create-aco-creds function access"

  security_group_id            = data.aws_security_group.db.id
  referenced_security_group_id = module.admin_create_aco_creds_function.security_group_id
}

resource "aws_vpc_security_group_egress_rule" "db_tcp" {
  from_port   = 5432
  to_port     = 5432
  ip_protocol = "tcp"
  description = "egress to db"

  security_group_id            = module.admin_create_aco_creds_function.security_group_id
  referenced_security_group_id = data.aws_security_group.db.id
}

resource "aws_vpc_security_group_egress_rule" "ssas_admin" {
  from_port   = 444
  to_port     = 444
  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "tcp"
  description = "egress to SSAS admin port"

  security_group_id = module.admin_create_aco_creds_function.security_group_id
}
