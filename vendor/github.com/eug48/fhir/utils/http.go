package utils

import (
	"fmt"
	"strings"
)


func ETagToVersionId(etag string) (string, error) {

	if strings.HasPrefix(etag, "W/&quot;") {
		etag = etag[8:]
	} else if strings.HasPrefix(etag, "W/\"") {
		etag = etag[3:]
	} else {
		return "", fmt.Errorf("ETag missing 'W/\"' prefix: %s", etag)
	}

	if strings.HasSuffix(etag, "\"") {
		etag = etag[:len(etag)-1]
	} else if strings.HasSuffix(etag, "&quot;") {
		etag = etag[:len(etag)-6]
	} else {
		return "", fmt.Errorf("ETag missing 'W/\"' suffix: %s", etag)
	}

	return etag, nil
}