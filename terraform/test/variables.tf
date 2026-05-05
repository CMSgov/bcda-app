variable "env" {
  default = "test"
}

variable "api_image_tag" {
  type        = string
  description = "Tag of the API image to deploy to the API ECS service"
}

variable "ssas_image_tag" {
  type        = string
  description = "Tag of the SSAS image to deploy to the SSAS ECS service"
}

variable "worker_image_tag" {
  type        = string
  description = "Tag of the worker image to deploy to the worker ECS service"
}

variable "ssl_policy" {
  type        = string
  description = "The AWS predefined SSL policy for the ALB"
  default     = "ELBSecurityPolicy-TLS13-1-2-Res-2021-06"
}

variable "kms_key_id" {
  type        = string
  default     = "1fca833c-6d08-44d2-8088-9be2c37155f7"
  description = "KMS Key ID for the ACO Creds KMS Key"
}

variable "eni_ips" {
  type        = list(string)
  description = "Private IP used to create the ENI"

  default = [
    "10.234.254.15",
  ]
}

variable "monitoring_interval" {
  default     = 60
  description = "The Aurora cluster enhanced monitoring interval (seconds)."
  type        = number
}

variable "log_retention_in_days" {
  default     = 30
  description = "How long to retain Cloudwatch logs."
  type        = number
}

