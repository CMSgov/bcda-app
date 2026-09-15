variable "cluster_name" {
  type = string
}

variable "service_name" {
  type = string
}

variable "ok_notification_arn" {
  type = string
} 

variable "alarm_notification_arn" {
  type = string
} 

variable "alarms" {
  type = list(object(
    { 
     alarm_name : string,
      alarm_description : string,
      metric_name : string,
      period : number, 
      eval_periods : number, 
      threshold : number }))
}
