package bcdaaws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

var s3Region = "us-east-1"

// Makes these easily mockable for testing
var newSession = session.NewSession
var newSessionWithOptions = session.NewSessionWithOptions

// NewSession
// Returns a new AWS session using the given roleArn
func NewSession(roleArn, endpoint string) (*session.Session, error) {
	sess := session.Must(session.NewSession())
	var err error

	config := aws.Config{
		Region: aws.String("us-east-1"),
	}

	if endpoint != "" {
		config.S3ForcePathStyle = aws.Bool(true)
		config.Endpoint = &endpoint
	}

	if roleArn != "" {
		config.Credentials = stscreds.NewCredentials(
			sess,
			roleArn,
		)
	}

	sess, err = newSession(&config)

	if err != nil {
		return nil, err
	}

	return sess, nil
}
