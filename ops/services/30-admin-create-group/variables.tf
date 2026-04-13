variable "app" {
  description = "The application name (bcda)"
  type        = string
  validation {
    condition     = contains(["bcda"], var.app)
    error_message = "Valid value for app is bcda."
  }
}

variable "env" {
  description = "The application environment (dev, test, sbx, sandbox, prod)"
  type        = string
  validation {
    condition     = contains(["dev", "test", "sbx", "sandbox", "prod"], var.env)
    error_message = "Valid value for env is dev, test, sbx, sandbox, or prod."
  }
}
