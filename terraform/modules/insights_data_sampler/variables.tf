# BFD variables
# We need to specify the ARN of the BFD S3 bucket KMS key because it lives in 
# BFDs account and we don't have permission to access it as a data source.
variable "bfd_bucket" {
  type        = string
  description = "The BFD bucket ARN where data will be streamed"
  default     = "arn:aws:s3:::bfd-insights-bcda-577373831711"
}

variable "bfd_kms_key" {
  type        = string
  description = "The Customer Managed Key (CMK) ARN used to encrypt the BFD S3 bucket"
  default     = "arn:aws:kms:us-east-1:577373831711:key/a4ca59d6-3978-434b-9a5c-c6ae8896db18"
}

# Cloudwatch variables
variable "name" {
  type        = string
  description = "Name of the quantity being sampled, e.g. num_jobs"
}

variable "description" {
  type        = string
  description = "description of what is being sampled and how often, e.g. Number of Jobs"
}

variable "schedule" {
  type        = string
  description = "Specified rate or cron expression"
}

variable "query" {
  type        = string
  description = "SQL query to pass to the database"
}

variable "env" {
  type        = string
  description = "The environment the data is coming from."
}

variable "database" {
  type        = string
  description = "The glue catalog database"
  default     = "bcda"
}

variable "db_conn_string_env_var" {
  type        = string
  description = "The variable name for the DB connection string.  This string does NOT reference an actual DB connection string; however, it references an env var in Parameter Store which maps to a connection string.  Most likely this value will be DATABASE_URL"
  default     = "DATABASE_URL"
}

variable "buffering_size" {
  default     = 5
  description = "Buffer incoming data to the specified size, in MBs, before delivering it to the destination."
}

variable "buffering_interval" {
  default     = 60
  description = "Buffer incoming data for the specified period of time, in seconds, before delivering it to the destination."
}

variable "insights_role_arn" {
  description = "The arn of the role to be associated with the aws_kinesis_firehose_delivery_stream s3 config"
}

variable "lambda_arn" {
  description = "The arn of the data sampler lambda function"
}
