package testUtils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

    configuration "github.com/CMSgov/bcda-app/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
)

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
	if err := configuration.SetEnv(&testing.T{}, key, value); err != nil {
		log.Printf("Error %s env value %s to %s\n", why, key, value)
	}
}

// SetAndRestoreEnvKey replaces the current value of the env var key,
// returning a function which can be used to restore the original value
func SetAndRestoreEnvKey(key, value string) func() {
	originalValue := configuration.GetEnv(key)
	setEnv("setting", key, value)
	return func() {
		setEnv("restoring", key, originalValue)
	}
}

// SetUnitTestKeysForAuth sets the env vars auth uses to locate its signing key pair. Intended for use only
// by unit tests of the API. Should be called by any API test that uses the auth backend. Returns a function
// that will restore the original values, suitable for use with defer.
func SetUnitTestKeysForAuth() func() {
	private := SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../shared_files/api_unit_test_auth_private.pem")
	public := SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../shared_files/api_unit_test_auth_public.pem")

	return func() {
		private()
		public()
	}
	// if these paths are incorrect, unit tests will fail with an unhelpful panic referencing a logrus entry
	// these paths only work because we assume that the binary location doesn't change from test to test or env to env
	// the issue, and a path to a more robust solution, is well described in
	// https://stackoverflow.com/questions/45579312/loading-a-needed-file-relative-vs-absolute-paths
}

func MakeDirToDelete(s suite.Suite, filePath string) {
	assert := assert.New(s.T())
	_, err := os.Create(filepath.Join(filePath, "deleteMe1.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(filePath, "deleteMe2.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(filePath, "deleteMe3.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(filePath, "deleteMe4.txt"))
	assert.Nil(err)
}

// SetPendingDeletionDir sets the PENDING_DELETION_DIR to the supplied "path" and ensures
// that the directory is created
func SetPendingDeletionDir(s suite.Suite, path string) {
	err := configuration.SetEnv(&testing.T{}, "PENDING_DELETION_DIR", path)
	if err != nil {
		s.FailNow("failed to set the PENDING_DELETION_DIR env variable,", err)
	}
	cclfDeletion := configuration.GetEnv("PENDING_DELETION_DIR")
	err = os.MkdirAll(cclfDeletion, 0744)
	if err != nil {
		s.FailNow("failed to create the pending deletion directory, %s", err.Error())
	}
}

// CopyToTemporaryDirectory copies all of the content found at src into a temporary directory.
// The path to the temporary directory is returned along with a function that can be called to clean up the data.
func CopyToTemporaryDirectory(t *testing.T, src string) (string, func()) {
	newPath, err := ioutil.TempDir("", "*")
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

// GetRandomIPV4Address returns a random IPV4 address using rand.Read() to generate the values.
func GetRandomIPV4Address(t *testing.T) string {
	data := make([]byte, 4)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err.Error())
	}

	return fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3])
}

// GetGormMock returns a gorm.DB along with a sqlmock instance used for testing
// This implementation is based off a newer version of GORM's postgres driver.
// See: https://github.com/go-gorm/postgres/blob/v1.0.5/postgres.go#L24
// In the newer versions, you can explicitly set the ConnPool on the postgres.Config struct.
// It allows the caller to inject sqlmock's db instance into gorm without forcing the caller to
// rely on connecting via DSN, which will always fail when using sqlmock.
func GetGormMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gdb, err := gorm.Open(nil, &gorm.Config{
		ConnPool: db,
	})
	if err != nil {
		t.Fatalf("Failed to instantiate gorm db %s", err.Error())
	}
	gdb.Dialector = &postgres.Dialector{}
	callbacks.RegisterDefaultCallbacks(gdb, &callbacks.Config{})

	return gdb, mock
}
