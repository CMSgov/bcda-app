package testUtils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/middleware"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// CtxMatcher allow us to validate that the caller supplied a context.Context argument
// See: https://github.com/stretchr/testify/issues/519
var CtxMatcher = mock.MatchedBy(func(ctx context.Context) bool { return true })

// PrintSeparator prints a line of stars to stdout
func PrintSeparator() {
	fmt.Println("**********************************************************************************")
}

func RandomHexID() string {
	b, err := someRandomBytes(4)
	if err != nil {
		return "not_a_random_client_id"
	}
	return fmt.Sprintf("%x", b)
}

// RandomMBI returns an 11 character string that represents an MBI
func RandomMBI(t *testing.T) string {
	b, err := someRandomBytes(6)
	assert.NoError(t, err)
	return fmt.Sprintf("%x", b)[0:11]
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func RandomBase64(n int) string {
	b, err := someRandomBytes(20)
	if err != nil {
		return "not_a_random_base_64_string"
	}
	return base64.StdEncoding.EncodeToString(b)
}

func setEnv(why, key, value string) {
	if err := conf.SetEnv(&testing.T{}, key, value); err != nil {
		log.Printf("Error %s env value %s to %s\n", why, key, value)
	}
}

// SetAndRestoreEnvKey replaces the current value of the env var key,
// returning a function which can be used to restore the original value
func SetAndRestoreEnvKey(key, value string) func() {
	originalValue := conf.GetEnv(key)
	setEnv("setting", key, value)
	return func() {
		setEnv("restoring", key, originalValue)
	}
}

func MakeDirToDelete(s suite.Suite, filePath string) {
	assert := assert.New(s.T())
	_, err := os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe1.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe2.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe3.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe4.txt")))
	assert.Nil(err)
}

// SetPendingDeletionDir sets the PENDING_DELETION_DIR to the supplied "path" and ensures
// that the directory is created
func SetPendingDeletionDir(s suite.Suite, path string) {
	err := conf.SetEnv(s.T(), "PENDING_DELETION_DIR", path)
	if err != nil {
		s.FailNow("failed to set the PENDING_DELETION_DIR env variable,", err)
	}
	cclfDeletion := conf.GetEnv("PENDING_DELETION_DIR")
	err = os.MkdirAll(cclfDeletion, 0744)
	if err != nil {
		s.FailNow("failed to create the pending deletion directory, %s", err.Error())
	}
}

// CopyToTemporaryDirectory copies all of the content found at src into a temporary directory.
// The path to the temporary directory is returned along with a function that can be called to clean up the data.
func CopyToTemporaryDirectory(t *testing.T, src string) (string, func()) {
	newPath, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory %s", err.Error())
	}

	if err = copy.Copy(src, newPath); err != nil {
		t.Fatalf("Failed to copy contents from %s to %s %s", src, newPath, err.Error())
	}

	cleanup := func() {
		err := os.RemoveAll(newPath)
		if err != nil {
			log.Printf("Failed to cleanup data %s", err.Error())
		}
	}

	return newPath, cleanup
}

// CopyToS3 copies all of the content found at src into a temporary S3 folder within localstack.
// The path to the temporary S3 directory is returned along with a function that can be called to clean up the data.
func CopyToS3(t *testing.T, src string) (string, func()) {
	tempBucket := uuid.NewUUID()

	endpoint := conf.GetEnv("BFD_S3_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for S3: %s", err.Error())
	}

	svc := s3.New(sess)

	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(tempBucket.String()),
	})

	if err != nil {
		t.Fatalf("Failed to create bucket %s: %s", tempBucket.String(), err.Error())
	}

	uploader := s3manager.NewUploader(sess)

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("Unexpected error reading path")
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(filepath.Clean(path))
		if err != nil {
			return err
		}

		key := path
		parts := strings.Split(path, "shared_files/")
		if len(parts) > 1 {
			key = parts[1]
		}

		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(tempBucket.String()),
			Key:    aws.String(key),
			Body:   f,
		})

		if err != nil {
			return err
		}

		fmt.Printf("Uploaded file in bucket %s, key %s\n", tempBucket.String(), key)
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to upload files to S3: %s", err.Error())
	}

	cleanup := func() {
		svc := s3.New(sess)
		iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
			Bucket: aws.String(tempBucket.String()),
		})

		// Traverse iterator deleting each object
		if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
			log.Printf("Unable to delete objects from bucket %s, %s\n", tempBucket, err)
		}
	}

	return tempBucket.String(), cleanup
}

type ZipInput struct {
	ZipName   string
	CclfNames []string
}

func CreateZipsInS3(t *testing.T, zipInputs ...ZipInput) (string, func()) {
	tempBucket := uuid.NewUUID()
	endpoint := conf.GetEnv("BFD_S3_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for S3: %s", err.Error())
	}

	svc := s3.New(sess)

	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(tempBucket.String()),
	})

	if err != nil {
		t.Fatalf("Failed to create bucket %s: %s", tempBucket.String(), err.Error())
	}

	for _, input := range zipInputs {
		var b bytes.Buffer
		f := bufio.NewWriter(&b)
		w := zip.NewWriter(f)

		for _, cclfName := range input.CclfNames {
			_, err := w.Create(cclfName)
			assert.NoError(t, err)
		}

		assert.NoError(t, w.Close())
		assert.NoError(t, f.Flush())

		uploader := s3manager.NewUploader(sess)

		_, s3Err := s3manager.Uploader.Upload(*uploader, &s3manager.UploadInput{
			Bucket: aws.String(tempBucket.String()),
			Key:    aws.String(input.ZipName),
			Body:   bytes.NewReader(b.Bytes()),
		})

		assert.NoError(t, s3Err)
	}

	cleanup := func() {
		svc := s3.New(sess)
		iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
			Bucket: aws.String(tempBucket.String()),
		})

		// Traverse iterator deleting each object
		if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
			logrus.Printf("Unable to delete objects from bucket %s, %s\n", tempBucket, err)
		}
	}

	return tempBucket.String(), cleanup
}

func ListS3Objects(t *testing.T, bucket string, prefix string) []*s3.Object {
	endpoint := conf.GetEnv("BFD_S3_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for S3: %s", err.Error())
	}

	svc := s3.New(sess)

	fmt.Printf("Listing objects in bucket %s, prefix %s", bucket, prefix)

	resp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		t.Fatalf("Failed to list objects in S3 bucket %s, prefix %s: %s", bucket, prefix, err)
	}

	return resp.Contents
}

// Inserts the provided parameter into localstack.
func PutParameter(t *testing.T, input *ssm.PutParameterInput) error {
	endpoint := conf.GetEnv("LOCAL_STACK_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for SSM: %s", err.Error())
	}

	fmt.Printf("Inserting parameter %s with value %s\n", *input.Name, *input.Value)

	svc := ssm.New(sess)
	_, err = svc.PutParameter(input)

	if err != nil {
		t.Fatalf("Failed to insert parameter %s with value %s: %s\n", *input.Name, *input.Value, err)
	}

	return nil
}

// Deletes the provided parameters from localstack.
func DeleteParameters(t *testing.T, input *ssm.DeleteParametersInput) error {
	endpoint := conf.GetEnv("LOCAL_STACK_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for SSM: %s", err.Error())
	}

	fmt.Printf("Deleting parameters from parameter store\n")

	svc := ssm.New(sess)
	_, err = svc.DeleteParameters(input)

	if err != nil {
		t.Fatalf("Failed to delete parameters: %s", err)
	}

	return nil
}

type AwsParameter struct {
	Name  string
	Value string
	Type  string
}

// Insert all given parameters into localstack and return a method for deferring cleanup.
func SetParameters(t *testing.T, params []AwsParameter) func() {
	var paramKeys []*string

	for _, paramInput := range params {
		err := PutParameter(t, &ssm.PutParameterInput{
			Name:  &paramInput.Name,
			Value: &paramInput.Value,
			Type:  &paramInput.Type,
		})

		assert.Nil(t, err)

		name := paramInput.Name
		paramKeys = append(paramKeys, &name)
	}

	cleanup := func() {
		for _, paramInput := range paramKeys {
			fmt.Printf("Deleting %s\n", *paramInput)
		}

		err := DeleteParameters(t, &ssm.DeleteParametersInput{Names: paramKeys})
		assert.Nil(t, err)
	}

	return cleanup
}

type EnvVar struct {
	Name  string
	Value string
}

// Update all given environment variables and return a method for deferring cleanup.
func SetEnvVars(t *testing.T, vars []EnvVar) func() {
	var origVars []EnvVar

	for _, envVar := range vars {
		origVars = append(origVars, EnvVar{Name: envVar.Name, Value: os.Getenv(envVar.Name)})
		os.Setenv(envVar.Name, envVar.Value)
	}

	cleanup := func() {
		for _, envVar := range origVars {
			os.Setenv(envVar.Name, envVar.Value)
		}
	}

	return cleanup
}

// GetRandomIPV4Address returns a random IPV4 address using rand.Read() to generate the values.
func GetRandomIPV4Address(t *testing.T) string {
	data := make([]byte, 3)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err.Error())
	}

	// Use static first byte to ensure that the IP address is valid
	return fmt.Sprintf("146.%d.%d.%d", data[0], data[1], data[2])
}

// GetLogger returns the underlying implementation of the field logger
func GetLogger(logger logrus.FieldLogger) *logrus.Logger {
	if entry, ok := logger.(*logrus.Entry); ok {
		return entry.Logger
	}
	// Must be a *logrus.Logger
	return logger.(*logrus.Logger)
}

// ReadResponseBody will read http.Response and return the body contents as a string.
func ReadResponseBody(r *http.Response) string {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		bodyString := fmt.Sprintf("Error reading the body: %s\n", err.Error())
		return bodyString
	}

	bodyString := bytes.NewBuffer(body).String()
	return bodyString
}

// MakeTestServerWithIntrospectEndpoint creates an httptest.Server with an introspect endpoint that will
// return back a response with a json body indicating if "active" is set to true or false (set by active
// token parameter)
func MakeTestServerWithIntrospectEndpoint(activeToken bool) *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.IntrospectPath, func(w http.ResponseWriter, r *http.Request) {
		var (
			buf   []byte
			input struct {
				Token string `json:"token"`
			}
		)
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Unexpected error creating test server: Error in reading request body: %s\n", err.Error())
			return
		}

		if unmarshalErr := json.Unmarshal(buf, &input); unmarshalErr != nil {
			fmt.Printf("Unexpected error creating test server: Error in unmarshalling the buffered input to JSON: %s\n", unmarshalErr.Error())
			return
		}

		body, _ := json.Marshal(struct {
			Active bool `json:"active"`
		}{Active: activeToken})

		_, responseWriterErr := w.Write(body)
		if responseWriterErr != nil {
			fmt.Printf("Unexpected error creating test server: Error reading request body: %s\n", responseWriterErr.Error())
		}

	})
	return httptest.NewServer(router)
}

// MakeTestServerWithIntrospectTimeout creates an httptest.Server with an introspect endpoint that will sleep for 10 seconds.
// Useful in testing where the env timeout is set to something less (ex. 5 seconds) and you want to ensure *url.Error.Timeout() returns true.
func MakeTestServerWithIntrospectTimeout() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.IntrospectPath, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 10)
	})

	return httptest.NewServer(router)
}

// MakeTestServerWithIntrospectReturn502 creates an httptest.Server
// with an introspect endpoint that will return 502 Status Code.
func MakeTestServerWithIntrospectReturn502() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.IntrospectPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithTokenRequestTimeout() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.TokenPath, func(w http.ResponseWriter, r *http.Request) {
		retrySeconds := strconv.FormatInt(int64(1), 10)
		w.Header().Set("Retry-After", retrySeconds)
		w.WriteHeader(http.StatusServiceUnavailable)
		time.Sleep(time.Second * 10)
	})

	return httptest.NewServer(router)
}

func MakeTestServerWithValidTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.TokenPath, func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "goodToken", "expires_in": "1200" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithInvalidCarriage() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.TokenPath+"\n", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "goodToken", "expires_in": "1200" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithBadRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.TokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`Bad Request`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithInvalidTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.TokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte(`Unauthorized`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithBadAuthTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.AuthTokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`Bad Request`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithAuthTokenRequestTimeout() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.AuthTokenPath, func(w http.ResponseWriter, r *http.Request) {
		retrySeconds := strconv.FormatInt(int64(1), 10)
		w.Header().Set("Retry-After", retrySeconds)
		w.WriteHeader(http.StatusServiceUnavailable)
		time.Sleep(time.Second * 10)
	})

	return httptest.NewServer(router)
}

func MakeTestServerWithValidAuthTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.AuthTokenPath, func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "token_type": "bearer", "access_token": "goodToken", "expires_in": "1200" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithInvalidAuthTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.AuthTokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte(`Unauthorized`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func MakeTestServerWithInternalServerErrAuthTokenRequest() *httptest.Server {
	router := chi.NewRouter()
	router.Post(constants.AuthTokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`Unexpected Error`))
		if err != nil {
			log.Fatal(err)
		}
	})
	return httptest.NewServer(router)
}

func ContextTransactionID() *http.Request {
	// this request url is a placeholder/arbitrary
	r := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/token", nil)
	ctx := context.Background()
	r = r.WithContext(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()))
	return r
}
