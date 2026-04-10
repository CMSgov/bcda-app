output "function_role_arn" {
  value = module.api_waf_sync_function.role_arn
}

output "zip_bucket" {
  value = module.api_waf_sync_function.zip_bucket
}
