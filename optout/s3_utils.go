package optout

import "strings"

// Parses an S3 URI and returns the bucket and key.
//
// @example:
//
//	input: s3://my-bucket/path/to/file
//	output: "my-bucket", "path/to/file"
//
// @example
//
//	input: s3://my-bucket
//	output: "my-bucket", ""
func ParseS3Uri(str string) (bucket string, key string) {
	workingString := strings.TrimPrefix(str, "s3://")
	resultArr := strings.SplitN(workingString, "/", 2)

	if len(resultArr) == 1 {
		return resultArr[0], ""
	}

	return resultArr[0], resultArr[1]
}
