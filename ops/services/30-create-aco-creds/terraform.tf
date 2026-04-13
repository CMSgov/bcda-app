provider "aws" {
  default_tags {
    tags = {
      application = var.app
      business    = "oeda"
      code        = "https://github.com/CMSgov/cdap/tree/main/terraform/services/admin-create-aco-creds"
      component   = "admin-create-aco-creds"
      environment = var.env
      terraform   = true
    }
  }
}

terraform {
  backend "s3" {
    key = "admin-create-aco-creds/terraform.tfstate"
  }
}
