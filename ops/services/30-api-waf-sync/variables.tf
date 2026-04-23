variable "env" {
  description = "The application environment (dev, test, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "test", "prod"], var.env)
    error_message = "Valid value for env is dev, test, or prod."
  }
}
