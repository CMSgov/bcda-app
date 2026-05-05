variable "env" {
  type        = string
  description = "The environment the data is coming from"
}

variable "db_subnet_group" {
  type        = string
  description = "The subnet group for the database"
}

variable "worker_security_group" {
  type        = string
  description = "The id of the worker security group"
}
