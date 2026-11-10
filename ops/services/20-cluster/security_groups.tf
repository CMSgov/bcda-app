# This file manages the security groups for the resources in the cluster.
# It does not manage security group RULES, which are handled by the services.

resource "aws_security_group" "app_sg" {
  name        = "bcda-api-${module.platform.env}"
  description = "bcda api app security group"
  vpc_id      = module.platform.vpc_id

  tags = {
    Name = "bcda-api-${module.platform.env}"
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      id,
      description,
    ]
  }
}

resource "aws_security_group" "sg_alb_01" {
  name        = "app-alb-${module.platform.env}"
  description = "App ALB security group"
  vpc_id      = module.platform.vpc_id

  tags = {
    "cms-cloud-exempt:open-sg" = local.sg_compliance_attestation_url
  }
}

resource "aws_security_group" "ssas_alb" {
  name        = "ssas-alb"
  description = "SSAS ALB security group"
  vpc_id      = module.platform.vpc_id
}

resource "aws_security_group" "worker_sg" {
  name        = "bcda-worker-${module.platform.env}"
  description = "bcda worker security group"
  vpc_id      = module.platform.vpc_id

  lifecycle {
    create_before_destroy = true
    ignore_changes = [
      id,
      description,
    ]
  }
}
