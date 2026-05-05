variable "env" {
  type = string
}

variable "tags" {
  type = map(string)
}

variable "s3_origin_id" {
  type = string
}

variable "static_site_domain_name" {
  type = string
}