variable "app" {
}

variable "service" {
}

variable "name" {
  description = "(Required) The reference_name of your file system. Also, used in tags."
  type        = string
}

variable "env" {
  description = "(Required) The environment of your file system. Also, used in tags."
  type        = string
}

variable "performance_mode" {
  description = "(Optional) The performance mode of your file system."
  type        = string
  default     = "generalPurpose"
}

variable "vpc_id" {
  description = "(Required) The VPC ID where EFS security groups will be."
  type        = string
}

variable "subnets" {
  description = "(Required) A comma separated list of subnet ids where mount targets will be."
  type        = list(string)
}

variable "additional_ingress_sgs" {
  type = list(string)
}

variable "kms_key_id" {
  description = "The ID of the KMS key used to encrypt the file system"
}

