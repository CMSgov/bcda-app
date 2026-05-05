variable "env" {
  description = "environment"
  type        = string
}

variable "db_subnet_group" {
  description = "Name of the database subnet group."
  type        = string
}

variable "security_group_id" {
  description = "Id of the security group to use for the lambda."
  type        = string
}
