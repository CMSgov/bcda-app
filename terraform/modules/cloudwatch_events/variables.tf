# AWS Cloudwatch Event Rule
variable "name" {
  type        = string
  description = "The name for the CloudWatch event rule function"
}

variable "description" {
  type        = string
  description = "The description for the CloudWatch event rule"
}

variable "schedule" {
  type        = string
  description = "CloudWatch Rate or Cron"
}

# AWS Cloudwatch Event Target
variable "target_id" {
  type        = string
  description = "(optional) Describe Event Target or assigned random"
}

variable "arn" {
  type        = string
  description = "ARN of AWS resource ex: aws_lambda_function.lambda.arn"
}

