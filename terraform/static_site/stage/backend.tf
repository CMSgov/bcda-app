terraform {
  backend "s3" {
    bucket       = "bcda-test-tfstate-20250409171646342600000001"
    key          = "static-site/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}
