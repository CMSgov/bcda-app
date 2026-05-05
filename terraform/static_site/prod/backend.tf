terraform {
  backend "s3" {
    bucket       = "bcda-prod-tfstate-20250411203841436200000001"
    key          = "static-site/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}
