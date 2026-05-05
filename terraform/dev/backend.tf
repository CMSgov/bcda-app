terraform {
  backend "s3" {
    bucket       = "bcda-dev-tfstate-20250409202710600700000001"
    key          = "dev/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}
