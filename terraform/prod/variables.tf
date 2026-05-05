variable "env" {
  default = "prod"
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

variable "eni_ips" {
  type        = list(string)
  description = "Private IP used to create the ENI"

  default = [
    "10.245.245.140",
  ]
}

variable "monitoring_interval" {
  default     = 15
  description = "The Aurora cluster enhanced monitoring interval (seconds)."
  type        = number
}

variable "log_retention_in_days" {
  default     = 30
  description = "How long to retain Cloudwatch logs."
  type        = number
}

