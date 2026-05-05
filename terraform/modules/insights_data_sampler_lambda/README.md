# Insights Lambda

### Description
Lambda function that samples the BCDA database and puts data in a firehose


### Deployment
Terraform will automatically deploy the packaged lambda function, but the packaging of the function source code itself must be performed manually. 

To deploy changes to the lambda source code:
1. Navigate to the sampler directory
`cd sampler`
2. Install application and download dependencies (requires node/npm)
`npm install`
3. Package lambda code as a zip in the lambda root directory
`zip -r ../insights_data_sampler.zip .`
4. Terraform plan/apply will deploy zip files changes to each environment
