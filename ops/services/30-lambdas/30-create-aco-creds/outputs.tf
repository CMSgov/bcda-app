output "function_role_arn" {
  value = module.admin_create_aco_creds_function.role_arn
}

output "zip_bucket" {
  value = module.admin_create_aco_creds_function.zip_bucket
}
