package models

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupTest() {
	InitializeGormModels()
	s.db = database.GetGORMDbConnection()
}

func (s *ModelsTestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *ModelsTestSuite) TestCreateACO() {
	assert := s.Assert()

	const ACOName = "ACO Name"
	cmsID := "A0000"
	acoUUID, err := CreateACO(ACOName, &cmsID)

	assert.Nil(err)
	assert.NotNil(acoUUID)

	var aco ACO
	err = s.db.Find(&aco, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.Equal(acoUUID.String(), aco.ClientID)
	assert.Equal(cmsID, *aco.CMSID)
	pubKey, err := aco.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(pubKey)
	assert.NotNil(GetATOPrivateKey())
	// should confirm the keys are a matched pair? i.e., encrypt something with one and decrypt with the other
	// the auth provider determines what the clientID contains (formatting, alphabet used, etc).
	// we require that it be representable in a string of less than 255 characters
	const ClientID = "Alpha client id"
	aco.ClientID = ClientID
	s.db.Save(aco)
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.NotNil(aco.ClientID)
	assert.Equal(ClientID, aco.ClientID)

	// make sure we can't duplicate the ACO UUID
	aco = ACO{
		UUID: acoUUID,
		Name: "Duplicate UUID Test",
	}
	err = s.db.Save(&aco).Error
	assert.EqualError(err, "pq: duplicate key value violates unique constraint \"acos_pkey\"")

	// Duplicate CMS ID
	aco = ACO{
		UUID:  uuid.NewRandom(),
		CMSID: &cmsID,
		Name:  "Duplicate CMS ID Test",
	}
	err = s.db.Save(&aco).Error
	assert.EqualError(err, "pq: duplicate key value violates unique constraint \"acos_cms_id_key\"")
}

func (s *ModelsTestSuite) TestACOPublicKeyColumn() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4444"
	aco := ACO{Name: "Pub Key Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	defer s.db.Delete(&aco)

	// Setup key
	pubKey := GetATOPublicKey()
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(pubKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save and verify
	aco.PublicKey = string(publicKeyBytes)
	err = s.db.Save(&aco).Error
	assert.Nil(err)
	err = s.db.First(&aco, "cms_id = ?", cmsID).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	assert.NotEmpty(aco.PublicKey)
	assert.Equal(publicKeyBytes, []byte(aco.PublicKey))
}

func (s *ModelsTestSuite) TestACOSavePublicKey() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4445"
	aco := ACO{Name: "Pub Key Save Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	defer s.db.Delete(&aco)

	// Setup key
	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err, "error creating random test keypair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save key
	err = aco.SavePublicKey(bytes.NewReader(publicKeyBytes))
	if err != nil {
		assert.FailNow("error saving key: " + err.Error())
	}

	// Retrieve and verify
	err = s.db.Find(&aco, "cms_id = ?", cmsID).Error
	assert.Nil(err, "unable to retrieve ACO from database")
	assert.NotNil(aco)
	assert.NotNil(aco.PublicKey)

	// Retrieve and verify
	storedKey, err := aco.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(storedKey)
	storedPublicKeyPKIX, err := x509.MarshalPKIXPublicKey(storedKey)
	assert.Nil(err, "unable to marshal saved public key")
	storedPublicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: storedPublicKeyPKIX,
	})
	assert.NotNil(storedPublicKeyBytes, "unexpectedly empty stored public key byte slice")
	assert.Equal(storedPublicKeyBytes, publicKeyBytes)
}

func (s *ModelsTestSuite) TestACOSavePublicKeyInvalidKey() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4447"
	aco := ACO{Name: "Pub Key Save Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	defer s.db.Delete(&aco)

	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	invalidPEM :=
		`-----BEGIN RSA PUBLIC KEY-----
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
OwIDAQAB
-----END RSA PUBLIC KEY-----`
	keyPair, err := rsa.GenerateKey(rand.Reader, 512)
	assert.Nil(err, "unable to generate key pair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	lowBitPubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(lowBitPubKey, "unexpectedly empty public key byte slice")

	err = aco.SavePublicKey(strings.NewReader(""))
	assert.NotNil(err, "empty string should not be saved")

	err = aco.SavePublicKey(strings.NewReader(emptyPEM))
	assert.NotNil(err, "empty PEM should not be saved")

	err = aco.SavePublicKey(strings.NewReader(invalidPEM))
	assert.NotNil(err, "invalid PEM should not be saved")

	err = aco.SavePublicKey(bytes.NewReader(lowBitPubKey))
	assert.NotNil(err, "insecure public key should not be saved")
}

func (s *ModelsTestSuite) TestACOPublicKeyEmpty() {
	assert := s.Assert()
	emptyPEM := "-----BEGIN RSA PUBLIC KEY-----    -----END RSA PUBLIC KEY-----"
	validPEM :=
		`-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
OwIDAQAB
-----END RSA PUBLIC KEY-----`
	emptyPubKey := ACO{PublicKey: ""}
	emptyPubKey2 := ACO{PublicKey: emptyPEM}
	nonEmptyPEM := ACO{PublicKey: validPEM}

	k, err := emptyPubKey.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(k, "Empty string does not yield nil public key!")
	k, err = emptyPubKey2.GetPublicKey()
	assert.EqualError(err, "not able to decode PEM-formatted public key")
	assert.Nil(k, "Empty PEM key does not yield nil public key!")
	k, err = nonEmptyPEM.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(k, "Valid PEM key yields nil public key!")
}

func (s *ModelsTestSuite) TestACOPublicKeyFixtures() {
	assert := s.Assert()
	acoUUID1 := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	acoUUID2 := constants.DevACOUUID

	var aco1 ACO
	var aco2 ACO
	err := s.db.First(&aco1, "uuid = ?", acoUUID1).Error
	assert.Nil(err)
	assert.NotEmpty(aco1, "This ACO (DBBD1CE1-AE24-435C-807D-ED45953077D3) is in the fixtures; why is it not being found?")
	assert.NotEmpty(aco1.PublicKey, "The fixture (DBBD1CE1-AE24-435C-807D-ED45953077D3) has data in the public_key column; why is it not being returned?")
	pubKey, err := aco1.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(pubKey, "Public key for DBBD1CE1-AE24-435C-807D-ED45953077D3 is unexpectedly nil.  Was there a parsing error in aco.GetPublicKey?")

	err = s.db.First(&aco2, "uuid = ?", acoUUID2).Error
	assert.Nil(err)
	assert.NotEmpty(aco2, "This ACO (0C527D2E-2E8A-4808-B11D-0FA06BAF8254) is in the fixtures; why is it not being found?")
	assert.NotEmpty(aco2.PublicKey, "The fixture (0C527D2E-2E8A-4808-B11D-0FA06BAF8254) has data in the public_key column; why is it not being returned?")
	pubKey, err = aco2.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(pubKey, "Public key for 0C527D2E-2E8A-4808-B11D-0FA06BAF8254 is unexpectedly nil.  Was there a parsing error in aco.GetPublicKey?")
}

func (s *ModelsTestSuite) TestACOPublicKeyRetrieve() {
	assert := s.Assert()

	// Setup ACO
	cmsID := "A4446"
	aco := ACO{Name: "Pub Key Test ACO", CMSID: &cmsID, UUID: uuid.NewRandom()}
	err := s.db.Create(&aco).Error
	assert.Nil(err)
	assert.NotEmpty(aco)
	defer s.db.Delete(&aco)

	// Setup key
	keyPair, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err, "error creating random test keypair")
	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	assert.Nil(err, "unable to marshal public key")
	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})
	assert.NotNil(publicKeyBytes, "unexpectedly empty public key byte slice")

	// Save
	aco.PublicKey = string(publicKeyBytes)
	err = s.db.Save(&aco).Error
	assert.Nil(err)
	s.db.Find(&aco, "cms_id = ?", cmsID)
	assert.NotNil(aco)
	assert.NotNil(aco.PublicKey)

	// Retrieve and verify
	storedKey, err := aco.GetPublicKey()
	if err != nil {
		assert.FailNow("error getting stored key")
	}
	if storedKey == nil {
		assert.FailNow("no stored key was found")
	}
	storedPublicKeyPKIX, err := x509.MarshalPKIXPublicKey(storedKey)
	assert.Nil(err, "unable to marshal saved public key")
	storedPublicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: storedPublicKeyPKIX,
	})
	assert.NotNil(storedPublicKeyBytes, "unexpectedly empty stored public key byte slice")
	assert.Equal(storedPublicKeyBytes, publicKeyBytes)
}

func (s *ModelsTestSuite) TestACOGetPublicKey_SSAS() {
	router := chi.NewRouter()
	keyStr := `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsZYpl2VjUja8VgkgoQ9K
lgjvcjwaQZ7pLGrIA/BQcm+KnCIYOHaDH15eVDKQ+M2qE4FHRwLec/DTqlwg8TkT
IYjBnXgN1Sg18y+SkSYYklO4cxlvMO3V8gaot9amPmt4YbpgG7CyZ+BOUHuoGBTh
z2v9wLlK4zPAs3pLln3R/4NnGFKw2Eku2JVFTotQ03gSmSzesZixicw8LxgYKbNV
oyTpERFansw6BbCJe7AP90rmaxCx80NiewFq+7ncqMbCMcqeUuCwk8MjS6bjvpcC
htFCqeRi6AAUDRg0pcG8yoM+jo13Z5RJPOIf3ofohncfH5wr5Q7qiOCE5VH4I7cp
OwIDAQAB
-----END RSA PUBLIC KEY-----
`
	router.Get("/system/{systemID}/key", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{ "client_id": "123456", "public_key": "` + strings.Replace(keyStr, "\n", "\\n", -1) + `" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	origAuthProvider := os.Getenv("BCDA_AUTH_PROVIDER")
	os.Setenv("BCDA_AUTH_PROVIDER", "ssas")
	defer os.Setenv("BCDA_AUTH_PROVIDER", origAuthProvider)

	origSSASURL := os.Getenv("SSAS_URL")
	os.Setenv("SSAS_URL", server.URL)
	defer os.Setenv("SSAS_URL", origSSASURL)

	origSSASUseTLS := os.Getenv("SSAS_USE_TLS")
	os.Setenv("SSAS_USE_TLS", "false")
	defer os.Setenv("SSAS_USE_TLS", origSSASUseTLS)

	cmsID := "A0001"
	aco := ACO{Name: "Public key from SSAS ACO", CMSID: &cmsID, UUID: uuid.NewRandom(), ClientID: "100"}

	key, err := aco.GetPublicKey()
	if err != nil {
		s.FailNow("Failed to get key", err.Error())
	}

	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		s.FailNow("Failed to marshal key", err.Error())
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: keyBytes,
	})

	assert.Equal(s.T(), keyStr, string(pemBytes))
}

func (s *ModelsTestSuite) TestCreateUser() {
	name, email, sampleUUID, duplicateName := "First Last", "firstlast@example.com", "DBBD1CE1-AE24-435C-807D-ED45953077D3", "Duplicate Name"

	// Make a user for an ACO that doesn't exist
	badACOUser, err := CreateUser(name, email, uuid.NewRandom())
	//No ID because it wasn't saved
	assert.True(s.T(), badACOUser.ID == 0)
	// Should get an error
	assert.NotNil(s.T(), err)

	// Make a good user
	user, err := CreateUser(name, email, uuid.Parse(sampleUUID))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), user.UUID)
	assert.NotNil(s.T(), user.ID)

	// Try making a duplicate user for the same E-mail address
	duplicateUser, err := CreateUser(duplicateName, email, uuid.Parse(sampleUUID))
	// Got a user, not the one that was requested
	assert.True(s.T(), duplicateUser.Name == name)
	assert.NotNil(s.T(), err)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) TestJobCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
	assert.Nil(s.T(), err)
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)
}
func (s *ModelsTestSuite) TestJobDefaultCompleted() {

	// Job is completed, but no keys exist.  This is fine, it is still complete
	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Completed",
		JobCount:   10,
	}
	s.db.Save(&j)

	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}
func (s *ModelsTestSuite) TestJobwithKeysCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   10,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	// JobKeys exist, but not enough to make the job complete
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	completed, err = j.CheckCompletedAndCleanup(s.db)
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}

func (s *ModelsTestSuite) TestGetEnqueJobs_AllResourcesTypes() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(constants.DevACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	enqueueJobs, err := j.GetEnqueJobs([]string{"Patient", "ExplanationOfBenefit", "Coverage"})
	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(3, len(enqueueJobs))
	var count = 0
	for _, queJob := range enqueueJobs {
		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		assert.Equal(int(j.ID), jobArgs.ID)
		assert.Equal(constants.DevACOUUID, jobArgs.ACOID)
		assert.Equal("6baf8254-2e8a-4808-b11d-0fa00c527d2e", jobArgs.UserID)
		if count == 0 {
			assert.Equal("Patient", jobArgs.ResourceType)
		} else if count == 1 {
			assert.Equal("ExplanationOfBenefit", jobArgs.ResourceType)
		} else {
			assert.Equal("Coverage", jobArgs.ResourceType)
		}
		assert.Equal(50, len(jobArgs.BeneficiaryIDs))
		count++
	}
}

func (s *ModelsTestSuite) TestGetEnqueJobs_Patient() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(constants.DevACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export?_type=Patient",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	enqueueJobs, err := j.GetEnqueJobs([]string{"Patient"})
	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(1, len(enqueueJobs))
	for _, queJob := range enqueueJobs {

		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		assert.Equal(int(j.ID), jobArgs.ID)
		assert.Equal(constants.DevACOUUID, jobArgs.ACOID)
		assert.Equal("6baf8254-2e8a-4808-b11d-0fa00c527d2e", jobArgs.UserID)
		assert.Equal("Patient", jobArgs.ResourceType)
		assert.Equal(50, len(jobArgs.BeneficiaryIDs))
	}
}

func (s *ModelsTestSuite) TestGetEnqueJobs_EOB() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(constants.DevACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export?_type=ExplanationOfBenefit",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	err := os.Setenv("BCDA_FHIR_MAX_RECORDS_EOB", "15")
	if err != nil {
		s.T().Error(err)
	}
	enqueueJobs, err := j.GetEnqueJobs([]string{"ExplanationOfBenefit"})
	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(4, len(enqueueJobs))
	enqueuedBenes := 0
	for _, queJob := range enqueueJobs {

		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		enqueuedBenes += len(jobArgs.BeneficiaryIDs)
		assert.True(len(jobArgs.BeneficiaryIDs) <= 15)
	}
	assert.Equal(50, enqueuedBenes)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_EOB")
}

func (s *ModelsTestSuite) TestGetEnqueJobs_Coverage() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(constants.DevACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export?_type=Coverage",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	err := os.Setenv("BCDA_FHIR_MAX_RECORDS_COVERAGE", "5")
	if err != nil {
		s.T().Error(err)
	}

	enqueueJobs, err := j.GetEnqueJobs([]string{"Coverage"})
	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(10, len(enqueueJobs))
	enqueuedBenes := 0
	for _, queJob := range enqueueJobs {

		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		enqueuedBenes += len(jobArgs.BeneficiaryIDs)
		assert.True(len(jobArgs.BeneficiaryIDs) <= 5)
	}
	assert.Equal(50, enqueuedBenes)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_COVERAGE")
}

func (s *ModelsTestSuite) TestJobStatusMessage() {
	j := Job{Status: "In Progress", JobCount: 25, CompletedJobCount: 6}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage())

	j = Job{Status: "In Progress", JobCount: 0, CompletedJobCount: 0}
	assert.Equal(s.T(), "In Progress", j.StatusMessage())

	j = Job{Status: "Completed", JobCount: 25, CompletedJobCount: 25}
	assert.Equal(s.T(), "Completed", j.StatusMessage())
}

func (s *ModelsTestSuite) TestGetMaxBeneCount() {
	assert := s.Assert()

	// ExplanationOfBenefit
	eobMax, err := GetMaxBeneCount("ExplanationOfBenefit")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT, eobMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_EOB", "5")
	if err != nil {
		s.T().Error(err)
	}
	eobMax, err = GetMaxBeneCount("ExplanationOfBenefit")
	assert.Equal(5, eobMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_EOB")

	// Patient
	patientMax, err := GetMaxBeneCount("Patient")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT, patientMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_PATIENT", "10")
	if err != nil {
		s.T().Error(err)
	}
	patientMax, err = GetMaxBeneCount("Patient")
	assert.Equal(10, patientMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_PATIENT")

	// Coverage
	coverageMax, err := GetMaxBeneCount("Coverage")
	assert.Equal(BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT, coverageMax)
	assert.Nil(err)

	err = os.Setenv("BCDA_FHIR_MAX_RECORDS_COVERAGE", "15")
	if err != nil {
		s.T().Error(err)
	}
	coverageMax, err = GetMaxBeneCount("Coverage")
	assert.Equal(15, coverageMax)
	assert.Nil(err)
	os.Unsetenv("BCDA_FHIR_MAX_RECORDS_COVERAGE")

	// Invalid type
	max, err := GetMaxBeneCount("Coverages")
	assert.Equal(-1, max)
	assert.EqualError(err, "invalid request type")
}

func (s *ModelsTestSuite) TestGetBeneficiaries() {
	assert := s.Assert()
	var aco, smallACO, mediumACO, largeACO ACO
	acoUUID := uuid.Parse(constants.DevACOUUID)

	err := s.db.Find(&aco, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaries, err := aco.GetBeneficiaries(true)
	assert.Nil(err)
	assert.NotNil(beneficiaries)
	assert.Equal(50, len(beneficiaries))

	// small ACO has 10 benes
	acoUUID = uuid.Parse(constants.SmallACOUUID)
	err = s.db.Debug().Find(&smallACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaries, err = smallACO.GetBeneficiaries(true)
	assert.Nil(err)
	assert.NotNil(beneficiaries)
	assert.Equal(10, len(beneficiaries))

	// Medium ACO has 25 benes
	acoUUID = uuid.Parse(constants.MediumACOUUID)
	err = s.db.Find(&mediumACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaries, err = mediumACO.GetBeneficiaries(true)
	assert.Nil(err)
	assert.NotNil(beneficiaries)
	assert.Equal(25, len(beneficiaries))

	// Large ACO has 100 benes
	acoUUID = uuid.Parse(constants.LargeACOUUID)
	err = s.db.Find(&largeACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaries, err = largeACO.GetBeneficiaries(true)
	assert.Nil(err)
	assert.NotNil(beneficiaries)
	assert.Equal(100, len(beneficiaries))

}

func (s *ModelsTestSuite) TestGetBeneficiaries_DuringETL() {
	acoCMSID := "T0000"
	aco := ACO{UUID: uuid.NewRandom(), CMSID: &acoCMSID}
	err := s.db.Save(&aco).Error
	if err != nil {
		s.FailNow("Failed to save ACO", err.Error())
	}
	defer s.db.Unscoped().Delete(&aco)

	cclfFile := CCLFFile{CCLFNum: 8, ACOCMSID: acoCMSID, Timestamp:time.Now().Add(-24 * time.Hour), ImportStatus:constants.ImportComplete}
	err = s.db.Save(&cclfFile).Error
	if err != nil {
		s.FailNow("Failed to save CCLF file", err.Error())
	}
	defer s.db.Unscoped().Delete(&cclfFile)

	bene1 := CCLFBeneficiary{FileID: cclfFile.ID, HICN: "bene1hicn"}
	err = s.db.Save(&bene1).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene1)

	// same aco newer file - in progress status
	cclfFileInProgress := CCLFFile{CCLFNum: 8, ACOCMSID: acoCMSID, Timestamp:time.Now(), ImportStatus:constants.ImportInprog}
	err = s.db.Save(&cclfFileInProgress).Error
	if err != nil {
		s.FailNow("Failed to save CCLF file", err.Error())
	}
	defer s.db.Unscoped().Delete(&cclfFileInProgress)

	bene2 := CCLFBeneficiary{FileID: cclfFileInProgress.ID, HICN: "bene2hicn"}
	err = s.db.Save(&bene2).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene2)

	// same aco newer file - failed status
	cclfFileFailed := CCLFFile{CCLFNum: 8, ACOCMSID: acoCMSID, Timestamp:time.Now(), ImportStatus:constants.ImportFail}
	err = s.db.Save(&cclfFileFailed).Error
	if err != nil {
		s.FailNow("Failed to save CCLF file", err.Error())
	}
	defer s.db.Unscoped().Delete(&cclfFileFailed)

	bene3 := CCLFBeneficiary{FileID: cclfFileFailed.ID, HICN: "bene2hicn"}
	err = s.db.Save(&bene3).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene3)

	result, err := aco.GetBeneficiaries(false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), result, 1)
	assert.Equal(s.T(),cclfFile.ID,result[0].FileID)
}

func (s *ModelsTestSuite) TestGetBeneficiaries_Unsuppressed() {
	acoCMSID := "T0000"
	aco := ACO{UUID: uuid.NewRandom(), CMSID: &acoCMSID}
	err := s.db.Save(&aco).Error
	if err != nil {
		s.FailNow("Failed to save ACO", err.Error())
	}
	defer s.db.Unscoped().Delete(&aco)

	cclfFile := CCLFFile{CCLFNum: 8, ACOCMSID: acoCMSID, ImportStatus:constants.ImportComplete}
	err = s.db.Save(&cclfFile).Error
	if err != nil {
		s.FailNow("Failed to save CCLF file", err.Error())
	}
	defer s.db.Unscoped().Delete(&cclfFile)

	// Beneficiary 1: preference indicator = N, effective date = now - 48 hours
	bene1 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene1_bbID"}
	err = s.db.Save(&bene1).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene1)

	bene1Suppression := Suppression{BlueButtonID: "bene1_bbID", PrefIndicator: "N", EffectiveDt: time.Now().Add(-48 * time.Hour)}
	err = s.db.Save(&bene1Suppression).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene1Suppression)

	// Beneficiary 2: preference indicator = Y, effective date = now - 24 hours
	bene2 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene2_bbID"}
	err = s.db.Save(&bene2).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene2)

	bene2Suppression := Suppression{BlueButtonID: "bene2_bbID", PrefIndicator: "Y", EffectiveDt: time.Now().Add(-24 * time.Hour)}
	err = s.db.Save(&bene2Suppression).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene2Suppression)

	// Beneficiary 3: no suppression record
	bene3 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene3_bbID"}
	err = s.db.Save(&bene3).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene3)

	// Beneficiary 4: preference indicator = N, effective date = now + 1 hour
	bene4 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene4_bbID"}
	err = s.db.Save(&bene4).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene4)

	bene4Suppression := Suppression{BlueButtonID: "bene4_bbID", PrefIndicator: "N", EffectiveDt: time.Now().Add(1 * time.Hour)}
	err = s.db.Save(&bene4Suppression).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene4Suppression)

	// Beneficiary 5: two suppression records, preference indicators = Y, N, effective dates = now - 72, 24 hours
	bene5 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene5_bbID"}
	err = s.db.Save(&bene5).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene5)

	bene5Suppression1 := Suppression{BlueButtonID: "bene5_bbID", PrefIndicator: "Y", EffectiveDt: time.Now().Add(-72 * time.Hour)}
	err = s.db.Save(&bene5Suppression1).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene5Suppression1)

	bene5Suppression2 := Suppression{BlueButtonID: "bene5_bbID", PrefIndicator: "N", EffectiveDt: time.Now().Add(-24 * time.Hour)}
	err = s.db.Save(&bene5Suppression2).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene5Suppression2)

	// Beneficiary 6: preference indicator = blank, effective date = now - 12 hours
	bene6 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene6_bbID"}
	err = s.db.Save(&bene6).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene6)

	bene6Suppression := Suppression{BlueButtonID: "bene6_bbID", PrefIndicator: "", EffectiveDt: time.Now().Add(-12 * time.Hour)}
	err = s.db.Save(&bene6Suppression).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene6Suppression)

	// Beneficiary 7: three suppression records: preference indicators = Y, N, Y; effective dates = now - 168, 96, 24 hours
	bene7 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene7_bbID"}
	err = s.db.Save(&bene7).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene7)

	bene7Suppression1 := Suppression{BlueButtonID: "bene7_bbID", PrefIndicator: "Y", EffectiveDt: time.Now().Add(-168 * time.Hour)}
	err = s.db.Save(&bene7Suppression1).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene7Suppression1)

	bene7Suppression2 := Suppression{BlueButtonID: "bene7_bbID", PrefIndicator: "N", EffectiveDt: time.Now().Add(-96 * time.Hour)}
	err = s.db.Save(&bene7Suppression2).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene7Suppression2)

	bene7Suppression3 := Suppression{BlueButtonID: "bene7_bbID", PrefIndicator: "Y", EffectiveDt: time.Now().Add(-24 * time.Hour)}
	err = s.db.Save(&bene7Suppression3).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene7Suppression3)

	// Beneficiary 8: three suppression records: preference indicators = Y, N, blank; effective dates = now - 96, 48, 24 hours
	bene8 := CCLFBeneficiary{FileID: cclfFile.ID, BlueButtonID: "bene8_bbID"}
	err = s.db.Save(&bene8).Error
	if err != nil {
		s.FailNow("Failed to save beneficiary", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene8)

	bene8Suppression1 := Suppression{BlueButtonID: "bene8_bbID", PrefIndicator: "Y", EffectiveDt: time.Now().Add(-96 * time.Hour)}
	err = s.db.Save(&bene8Suppression1).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene8Suppression1)

	bene8Suppression2 := Suppression{BlueButtonID: "bene8_bbID", PrefIndicator: "N", EffectiveDt: time.Now().Add(-48 * time.Hour)}
	err = s.db.Save(&bene8Suppression2).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene8Suppression2)

	bene8Suppression3 := Suppression{BlueButtonID: "bene8_bbID", PrefIndicator: "", EffectiveDt: time.Now().Add(-24 * time.Hour)}
	err = s.db.Save(&bene8Suppression3).Error
	if err != nil {
		s.FailNow("Failed to save suppression", err.Error())
	}
	defer s.db.Unscoped().Delete(&bene8Suppression3)

	result, err := aco.GetBeneficiaries(false)
	assert.Nil(s.T(), err)
	assert.Len(s.T(), result, 5)
	assert.Equal(s.T(), bene2.ID, result[0].ID)
	assert.Equal(s.T(), bene3.ID, result[1].ID)
	assert.Equal(s.T(), bene4.ID, result[2].ID)
	assert.Equal(s.T(), bene6.ID, result[3].ID)
	assert.Equal(s.T(), bene7.ID, result[4].ID)
}

func (s *ModelsTestSuite) TestGetBlueButtonID_CCLFBeneficiary() {
	assert := s.Assert()
	cclfBeneficiary := CCLFBeneficiary{HICN: "HASH_ME", MBI: "NOTHING"}
	bbc := testUtils.BlueButtonClient{}
	bbc.HICN = &cclfBeneficiary.HICN
	bbc.On("GetPatientByHICNHash", client.HashHICN(cclfBeneficiary.HICN)).Return(bbc.GetData("Patient", "BB_VALUE"))
	db := database.GetGORMDbConnection()
	defer db.Close()

	// New never seen before hicn, asks the mock blue button client for the value
	blueButtonID, err := cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// trivial case.  The object has a BB ID set on it already, this does nothing
	cclfBeneficiary.BlueButtonID = "LOCAL_VAL"
	blueButtonID, err = cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("LOCAL_VAL", blueButtonID)

	// Should be making only a single call to BB for all 2 attempts.
	bbc.AssertNumberOfCalls(s.T(), "GetPatientByHICNHash", 1)
}

func (s *ModelsTestSuite) TestGetBlueButtonID_Suppression() {
	assert := s.Assert()
	suppressBene := Suppression{HICN: "HASH_ME"}
	bbc := testUtils.BlueButtonClient{}
	bbc.HICN = &suppressBene.HICN
	bbc.On("GetPatientByHICNHash", client.HashHICN(suppressBene.HICN)).Return(bbc.GetData("Patient", "BB_VALUE"))
	db := database.GetGORMDbConnection()
	defer db.Close()

	// New never seen before hicn, asks the mock blue button client for the value
	blueButtonID, err := suppressBene.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// Should be making only a single call to BB for all 2 attempts.
	bbc.AssertNumberOfCalls(s.T(), "GetPatientByHICNHash", 1)
}
