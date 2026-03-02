variable "create_local_sops_wrapper" {
  default     = false
  description = "When `true`, creates sops wrapper file at `bin/sopsw`."
  type        = bool
}
