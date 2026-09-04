data "aws_security_group" "db" {
  name = local.db_sg_name
}

resource "aws_vpc_security_group_egress_rule" "ssas_admin" {
  from_port   = 444
  to_port     = 444
  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "tcp"
  description = "egress to SSAS admin port"

  security_group_id = module.rotate_ssas_creds_function.security_group_id
}
