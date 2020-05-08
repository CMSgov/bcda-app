package testUtils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PrintSeparator prints a line of stars to stdout
func PrintSeparator() {
	fmt.Println("**********************************************************************************")
}

func CreateStaging(jobID string) {
	err := os.Setenv("FHIR_STAGING_DIR", "data/test")
	if err != nil {
		log.Panic(err)
	}
	testdir := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)

	if _, err := os.Stat(testdir); os.IsNotExist(err) {
		err = os.MkdirAll(testdir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
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
	if err := os.Setenv(key, value); err != nil {
		log.Printf("Error %s env value %s to %s\n", why, key, value)
	}
}

// SetAndRestoreEnvKey replaces the current value of the env var key,
// returning a function which can be used to restore the original value
func SetAndRestoreEnvKey(key, value string) func() {
	originalValue := os.Getenv(key)
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
	dirToDelete := filePath
	err := os.Mkdir(dirToDelete, os.ModePerm)
	assert.Nil(err)

	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe1.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe2.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe3.txt"))
	assert.Nil(err)
	_, err = os.Create(filepath.Join(dirToDelete, "deleteMe4.txt"))
	assert.Nil(err)
}

func SetPendingDeletionDir(s suite.Suite) {
	pendingDeletionDirKey := "PENDING_DELETION_DIR"
	err := os.Setenv(pendingDeletionDirKey, "/go/src/github.com/CMSgov/bcda-app/bcda/pending_delete_dir")
	if err != nil {
		s.FailNow("failed to set the PENDING_DELETION_DIR env variable,", err)
	}
	cclfDeletion := os.Getenv(pendingDeletionDirKey)
	err = os.MkdirAll(cclfDeletion, 0744)
	if err != nil {
		s.FailNow("failed to create the pending deletion directory,", err)
	}
}

func ResetFiles(s suite.Suite, resetPath string) {
	err := filepath.Walk(os.Getenv("PENDING_DELETION_DIR"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				s.FailNow("error in walkfunc,", err)
			}

			if info.IsDir() {
				return nil
			}
			err = os.Rename(path, resetPath+info.Name())
			if err != nil {
				s.FailNow("error in moving files,", err)
			}
			return nil
		})
	if err != nil {
		s.FailNow("error in walkfunc,", err)
	}
}
