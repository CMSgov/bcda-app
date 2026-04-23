output "function_role_arn" {
  value = module.admin_create_aco_function.role_arn
}

output "zip_bucket" {
  value = module.admin_create_aco_function.zip_bucket
}
