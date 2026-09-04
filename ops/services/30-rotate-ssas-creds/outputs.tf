output "function_role_arn" {
  value = module.rotate_ssas_creds_function.role_arn
}

output "zip_bucket" {
  value = module.rotate_ssas_creds_function.zip_bucket
}
