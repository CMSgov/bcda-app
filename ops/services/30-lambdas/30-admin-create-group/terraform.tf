provider "aws" {
  default_tags {
    tags = {
      application = var.app
      business    = "oeda"
      code        = "https://github.com/CMSgov/cdap/tree/main/terraform/services/admin-create-group"
      component   = "admin-create-group"
      environment = var.env
      terraform   = true
    }
  }
}

terraform {
  backend "s3" {
    key = "admin-create-group/terraform.tfstate"
  }
}
