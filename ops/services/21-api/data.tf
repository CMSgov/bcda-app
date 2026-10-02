data "aws_vpc" "main" {
  id = module.platform.vpc_id
}

data "aws_security_group" "app_sg" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["bcda-api-${module.platform.env}"]
  }
}

data "aws_security_group" "ssas_alb" {
  vpc_id = module.platform.vpc_id
  filter {
    name   = "group-name"
    values = ["ssas-alb"]
  }
}
