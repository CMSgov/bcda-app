output "lambda_arn" {
  value = aws_lambda_function.insights_data_sampler.arn
}

output "lambda_function_name" {
  value = aws_lambda_function.insights_data_sampler.function_name
}