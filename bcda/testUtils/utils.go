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
	"math"
	"math/big"
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
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"

	"github.com/ccoveille/go-safecast"
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

func MakeDirToDelete(s *suite.Suite, filePath string) {
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
func SetPendingDeletionDir(s *suite.Suite, path string) {
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

func TestAWSConfig(t *testing.T) aws.Config {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(constants.DefaultRegion),
	)
	assert.Nil(t, err)

	return cfg
}

func TestS3Client(t *testing.T, cfg aws.Config) *s3.Client {
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for localstack buckets
	})
}

func TestSSMClient(t *testing.T, cfg aws.Config) *ssm.Client {
	return ssm.NewFromConfig(cfg)
}

// CopyToS3 copies all of the content found at src into a temporary S3 folder within localstack.
// The path to the temporary S3 directory is returned along with a function that can be called to clean up the data.
func CopyToS3(t *testing.T, src string) (string, func()) {
	ctx := context.Background()
	tempBucket := uuid.NewUUID().String()

	client := TestS3Client(t, TestAWSConfig(t))

	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(tempBucket),
	}
	_, err := client.CreateBucket(ctx, bucketInput)
	assert.Nil(t, err)

	if err != nil {
		t.Fatalf("Failed to create bucket %s: %s", tempBucket, err.Error())
	}

	uploader := manager.NewUploader(client)

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

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(tempBucket),
			Key:    aws.String(key),
			Body:   f,
		})

		if err != nil {
			return err
		}

		fmt.Printf("Uploaded file in bucket %s, key %s\n", tempBucket, key)
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to upload files to S3: %s", err.Error())
	}

	cleanup := func() {
		output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(tempBucket),
		})
		assert.Nil(t, err)

		var objIds []types.ObjectIdentifier
		for _, obj := range output.Contents {
			objIds = append(objIds, types.ObjectIdentifier{Key: obj.Key})
		}
		if len(objIds) > 0 {
			input := s3.DeleteObjectsInput{
				Bucket: aws.String(tempBucket),
				Delete: &types.Delete{
					Objects: objIds,
					Quiet:   aws.Bool(true),
				},
			}
			_, err = client.DeleteObjects(ctx, &input)
			assert.Nil(t, err)
		}
	}

	return tempBucket, cleanup
}

type ZipInput struct {
	ZipName   string
	CclfNames []string
}

func CreateZipsInS3(t *testing.T, zipInputs ...ZipInput) (string, func()) {
	ctx := context.Background()
	tempBucket := uuid.NewUUID().String()

	client := TestS3Client(t, TestAWSConfig(t))

	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String(tempBucket),
	}
	_, err := client.CreateBucket(ctx, bucketInput)
	assert.Nil(t, err)

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

		uploader := manager.NewUploader(client)

		_, s3Err := uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(tempBucket),
			Key:    aws.String(input.ZipName),
			Body:   bytes.NewReader(b.Bytes()),
		})

		assert.NoError(t, s3Err)
	}

	cleanup := func() {
		output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(tempBucket),
		})
		assert.Nil(t, err)

		var objIds []types.ObjectIdentifier
		for _, obj := range output.Contents {
			objIds = append(objIds, types.ObjectIdentifier{Key: obj.Key})
		}

		if len(objIds) > 0 {
			input := s3.DeleteObjectsInput{
				Bucket: aws.String(tempBucket),
				Delete: &types.Delete{
					Objects: objIds,
					Quiet:   aws.Bool(true),
				},
			}
			_, err = client.DeleteObjects(ctx, &input)
			assert.Nil(t, err)
		}
	}

	return tempBucket, cleanup
}

// Inserts the provided parameter into localstack.
func putParameter(t *testing.T, input ssm.PutParameterInput) error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(constants.DefaultRegion),
	)
	assert.Nil(t, err)
	client := ssm.NewFromConfig(cfg)

	_, err = client.PutParameter(ctx, &input)
	assert.Nil(t, err)

	return nil
}

// Deletes the provided parameters from localstack.
func deleteParameters(t *testing.T, input ssm.DeleteParametersInput) error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(constants.DefaultRegion),
	)
	assert.Nil(t, err)
	client := ssm.NewFromConfig(cfg)

	_, err = client.DeleteParameters(ctx, &input)
	assert.Nil(t, err)

	return nil
}

// Insert all given parameters into localstack and return a method for deferring cleanup.
func SetParameter(t *testing.T, name, value string) func() {
	err := putParameter(t, ssm.PutParameterInput{
		Name:  &name,
		Value: &value,
		Type:  "String",
	})
	assert.Nil(t, err)

	cleanup := func() {
		err := deleteParameters(t, ssm.DeleteParametersInput{Names: []string{name}})
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
		err := os.Setenv(envVar.Name, envVar.Value)
		assert.Nil(t, err)
	}

	cleanup := func() {
		for _, envVar := range origVars {
			err := os.Setenv(envVar.Name, envVar.Value)
			assert.Nil(t, err)
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

func GetSQSEvent(t *testing.T, bucketName string, fileName string) events.SQSEvent {
	jsonFile, err := os.Open("../../../shared_files/aws/s3event.json")
	if err != nil {
		fmt.Println(err)
	}

	defer func() {
		err := jsonFile.Close()
		assert.Nil(t, err)
	}()

	byteValue, err := io.ReadAll(jsonFile)
	assert.Nil(t, err)

	var s3event events.S3Event
	err = json.Unmarshal([]byte(byteValue), &s3event)
	assert.Nil(t, err)

	s3event.Records[0].S3.Bucket.Name = bucketName
	s3event.Records[0].S3.Object.Key = fileName

	val, err := json.Marshal(s3event)
	assert.Nil(t, err)

	body := fmt.Sprintf("{\"Type\" : \"Notification\",\n  \"MessageId\" : \"123456-1234-1234-1234-6e06896db643\",\n  \"TopicArn\" : \"my-topic\",\n  \"Subject\" : \"Amazon S3 Notification\",\n  \"Message\" : %s}", strconv.Quote(string(val[:])))
	event := events.SQSEvent{
		Records: []events.SQSMessage{{Body: body}},
	}
	return event
}

func GetFileFromZip(t *testing.T, zipReader *zip.Reader, filename string) *zip.File {
	for _, f := range zipReader.File {
		if f.Name == filename {
			return f
		}
	}

	return nil
}

func CryptoRandInt31() int32 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<31))
	if err != nil {
		panic(err) // handle error appropriately
	}

	o, _ := safecast.ToInt32(n.Int64())
	return o
}

func CryptoRandInt63() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		panic(err)
	}
	return n.Int64()
}
