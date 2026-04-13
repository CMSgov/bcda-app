output "function_role_arn" {
  value = module.admin_aco_deny_function.role_arn
}

output "zip_bucket" {
  value = module.admin_aco_deny_function.zip_bucket
}
