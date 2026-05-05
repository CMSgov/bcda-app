output "insights_role_arn" {
  value = aws_iam_role.insights_role.arn
}

output "lambda_arn" {
  value = module.insights_data_sampler.lambda_arn
}
