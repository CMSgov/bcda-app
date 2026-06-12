data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_vpc_security_group_ingress_rule" "function_db_access" {
  from_port   = 5432
  to_port     = 5432
  ip_protocol = "tcp"
  description = "admin-create-group function access"

  security_group_id            = data.aws_security_group.db.id
  referenced_security_group_id = module.admin_create_group_function.security_group_id
  depends_on                   = [aws_security_group_rule.function_access] # TODO: Delete depends_on after deploying BCDA-10031
}

resource "aws_vpc_security_group_egress_rule" "db_tcp" {
  from_port   = 5432
  to_port     = 5432
  ip_protocol = "tcp"
  description = "egress to db"

  security_group_id            = module.admin_create_group_function.security_group_id
  referenced_security_group_id = data.aws_security_group.db.id
}
