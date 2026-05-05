variable "env" {
  type        = string
  description = "The environment for the bucket"
}

variable "name" {
  type        = string
  description = "Name prefix for S3 bucket"
}

variable "extra_bucket_policies" {
  type        = list(string)
  default     = []
  description = "Extra bucket policies to merge with the existing policy"
}
