package bcdaaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
)

var s3Region = "us-east-1"
var DefaultRegion = "us-east-1"

// Makes these easily mockable for testing
// var newSession = session.NewSession

// NewSession
// Returns a new AWS session using the given roleArn
// func NewSession(roleArn, endpoint string) (*session.Session, error) {
// 	sess := session.Must(session.NewSession())
// 	var err error

// 	config := aws.Config{
// 		Region: aws.String(s3Region),
// 	}

// 	if endpoint != "" {
// 		config.S3ForcePathStyle = aws.Bool(true)
// 		config.Endpoint = &endpoint
// 	}

// 	if roleArn != "" {
// 		config.Credentials = stscreds.NewCredentials(
// 			sess,
// 			roleArn,
// 		)
// 	}

// 	sess, err = newSession(&config)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return sess, nil
// }

func NewAWSConfig(ctx context.Context, roleArn, endpoint string) (config.Config, error) {
	return config.LoadDefaultConfig(ctx, config.WithRegion(DefaultRegion))
	// var cfg config.Config

	// // used to override for localstack
	// if endpoint != "" {
	// 	cfg = config.LoadDefaultConfig(
	// 		ctx,
	// 		config.WithRegion(DefaultRegion)),

	// 	// cfg.S3ForcePathStyle = true
	// 	// cfg.Endpoint = &endpoint
	// } else if roleArn != "" {

	// 	client := stscreds.NewFromConfig(cfg)
	// 	appCreds := stscreds.NewAssumeRoleProvider(client, roleArn)
	// 	creds, err := appCreds.Retrieve(ctx)
	// 	if err != nil {
	// 		return config.Config{}, err
	// 	}

	// 	cfg.Credentials = creds
	// } else {

	// }

	// return cfg, nil
}
