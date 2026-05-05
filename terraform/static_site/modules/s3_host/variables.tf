variable "env" {
  type = string
}

variable "tags" {
  type = map(string)
}

variable "versioning" {
  description = "Enable Object Versioning"
  default     = true
}

variable "static_site_domain_name" {
  type = string
}