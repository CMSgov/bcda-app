# This file manages the security groups for the resources in the cluster.
# It does not manage security group RULES, which are handled by the services.

resource "aws_security_group" "ssas_alb" {
  name        = "ssas-alb"
  description = "SSAS ALB security group"
  vpc_id      = module.platform.vpc_id
}
