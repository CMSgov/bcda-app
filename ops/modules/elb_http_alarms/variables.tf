variable "vpc_name" {
  description = "Name of the VPC these alarms are for."
  type        = string
}

variable "load_balancer_name" {
  description = "Name of the ELB these alarms are for."
  type        = string
}

variable "cloudwatch_notification_arn" {
  description = "The CloudWatch notification ARN."
  type        = string
}

variable "app" {
}

variable "env" {
}

variable "alarm_elb_no_backend_enable" {
  default = true
}

variable "alarm_elb_no_backend_treat_missing_data" {
  default = "breaching"
}

variable "alarm_elb_no_backend_eval_periods" {
  default = 1
}

variable "alarm_elb_no_backend_period" {
  default = 60
}

variable "alarm_elb_no_backend_threshold" {
  default = 1
}

variable "alarm_elb_high_latency_enable" {
  default = true
}

variable "alarm_elb_high_latency_treat_missing_data" {
  default = "breaching"
}

variable "alarm_elb_high_latency_eval_periods" {
  default = 1
}

variable "alarm_elb_high_latency_period" {
  default = 900
}

variable "alarm_elb_high_latency_threshold" {
  default = 10
}

variable "alarm_elb_spillover_count_enable" {
  default = true
}

variable "alarm_elb_spillover_count_eval_periods" {
  default = 1
}

variable "alarm_elb_spillover_count_period" {
  default = 60
}

variable "alarm_elb_spillover_count_threshold" {
  default = 3
}

variable "alarm_elb_surge_queue_length_enable" {
  default = true
}

variable "alarm_elb_surge_queue_length_eval_periods" {
  default = 3
}

variable "alarm_elb_surge_queue_length_period" {
  default = 60
}

variable "alarm_elb_surge_queue_length_threshold" {
  default = 300
}

variable "alarm_backend_4xx_enable" {
  default = true
}

variable "alarm_backend_4xx_eval_periods" {
  default = 1
}

variable "alarm_backend_4xx_period" {
  default = 60
}

variable "alarm_backend_4xx_threshold" {
  default = 1
}

variable "alarm_backend_5xx_enable" {
  default = true
}

variable "alarm_backend_5xx_eval_periods" {
  default = 1
}

variable "alarm_backend_5xx_period" {
  default = 60
}

variable "alarm_backend_5xx_threshold" {
  default = 1
}

variable "alarm_elb_5xx_enable" {
  default = true
}

variable "alarm_elb_5xx_eval_periods" {
  default = 1
}

variable "alarm_elb_5xx_period" {
  default = 60
}

variable "alarm_elb_5xx_threshold" {
  default = 1
}

