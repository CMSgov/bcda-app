provider "aws" {
  default_tags {
    tags = {
      application = var.app
      business    = "oeda"
      code        = "https://github.com/CMSgov/cdap/tree/main/terraform/services/admin-aco-deny"
      component   = "admin-aco-deny"
      environment = var.env
      terraform   = true
    }
  }
}

terraform {
  backend "s3" {
    key = "admin-aco-deny/terraform.tfstate"
  }
}
