terraform {
  backend "s3" {
    bucket       = "bcda-sandbox-tfstate-20250416201512973800000001"
    key          = "sandbox/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}
