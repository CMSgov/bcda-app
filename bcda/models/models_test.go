package models

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
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
	assert.Equal("", aco.ClientID)
	assert.Equal(cmsID, *aco.CMSID)
	pubKey, err := aco.GetPublicKey()
	assert.NotNil(err)
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
	assert.NotNil(err)

	// Duplicate CMS ID
	aco = ACO{
		UUID:  uuid.NewRandom(),
		CMSID: &cmsID,
		Name:  "Duplicate CMS ID Test",
	}
	err = s.db.Save(&aco).Error
	assert.NotNil(err)
}

func (s *ModelsTestSuite) TestACOPublicKeySave() {
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
	assert.NotNil(err)
	assert.Nil(k, "Empty string does not yield nil public key!")
	k, err = emptyPubKey2.GetPublicKey()
	assert.NotNil(err)
	assert.Nil(k, "Empty PEM key does not yield nil public key!")
	k, err = nonEmptyPEM.GetPublicKey()
	assert.Nil(err)
	assert.NotNil(k, "Valid PEM key yields nil public key!")
}

func (s *ModelsTestSuite) TestACOPublicKeyFixtures() {
	assert := s.Assert()
	acoUUID1 := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	acoUUID2 := "0C527D2E-2E8A-4808-B11D-0FA06BAF8254"

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
	cmsID := "A4445"
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
	completed, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
	assert.Nil(s.T(), err)
	completed, err = j.CheckCompletedAndCleanup()
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

	completed, err := j.CheckCompletedAndCleanup()
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
	completed, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	// JobKeys exist, but not enough to make the job complete
	completed, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	completed, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}

func (s *ModelsTestSuite) TestGetEnqueJobs() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(constants.DEVACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	enqueueJobs, err := j.GetEnqueJobs("Patient")

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
		assert.Equal(constants.DEVACOUUID, jobArgs.ACOID)
		assert.Equal("6baf8254-2e8a-4808-b11d-0fa00c527d2e", jobArgs.UserID)
		assert.Equal("Patient", jobArgs.ResourceType)
		assert.Equal(50, len(jobArgs.BeneficiaryIDs))
	}

	j = Job{
		ACOID:      uuid.Parse(constants.DEVACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Pending",
	}

	s.db.Save(&j)
	defer s.db.Delete(&j)
	os.Setenv("BCDA_FHIR_MAX_RECORDS", "15")

	enqueueJobs, err = j.GetEnqueJobs("ExplanationOfBenefit")
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

}

func (s *ModelsTestSuite) TestGetBeneficiaryIDs() {
	assert := s.Assert()
	var aco, smallACO, mediumACO, largeACO ACO
	acoUUID := uuid.Parse(constants.DEVACOUUID)

	err := s.db.Find(&aco, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err := aco.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(50, len(beneficiaryIDs))

	// small ACO has 10 benes
	acoUUID = uuid.Parse(constants.SMALLACOUUID)
	err = s.db.Debug().Find(&smallACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = smallACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(10, len(beneficiaryIDs))

	// Medium ACO has 25 benes
	acoUUID = uuid.Parse(constants.MEDIUMACOUUID)
	err = s.db.Find(&mediumACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = mediumACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(25, len(beneficiaryIDs))

	// Large ACO has 100 benes
	acoUUID = uuid.Parse(constants.LARGEACOUUID)
	err = s.db.Find(&largeACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = largeACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(100, len(beneficiaryIDs))

}

func (s *ModelsTestSuite) TestGroupModel() {
	groupBytes := []byte(`{  
		"group_id":"A12345",
		"data":{  
			"name":"ACO Corp Systems",
			"users":[  
				"00uiqolo7fEFSfif70h7",
				"l0vckYyfyow4TZ0zOKek",
				"HqtEi2khroEZkH4sdIzj"
			],
			"scopes":[  
				"user-admin",
				"system-admin"
			],
			"resources":[  
				{  
					"id":"xxx",
					"name":"BCDA API",
					"scopes":[  
						"bcda-api"
					]
				},
				{  
					"id":"eft",
					"name":"EFT CCLF",
					"scopes":[  
						"eft-app:download",
						"eft-data:read"
					]
				}
			],
			"systems":[  
				{  
					"client_id":"4tuhiOIFIwriIOH3zn",
					"software_id":"4NRB1-0XZABZI9E6-5SM3R",
					"client_name":"ACO System A",
					"client_uri":"https://www.acocorpsite.com"
				}
			]
		}
	}`)

	group := Group{}
	err := json.Unmarshal(groupBytes, &group)
	assert.Nil(s.T(), err)
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Save(&group).Error
	assert.Nil(s.T(), err)
}

func (s *ModelsTestSuite) TestEncryptionKeyModel() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	group := Group{GroupID: "A00000"}
	db.Save(&group)

	system := System{GroupID: "A00000"}
	db.Save(&system)

	systemIDStr := strconv.FormatUint(uint64(system.ID), 10)
	encryptionKeyBytes := []byte(`{"body": "this is a public key", "system_id": ` + systemIDStr + `}`)
	encryptionKey := EncryptionKey{}
	err := json.Unmarshal(encryptionKeyBytes, &encryptionKey)
	assert.Nil(s.T(), err)

	err = db.Save(&encryptionKey).Error
	assert.Nil(s.T(), err)

	s.db.Unscoped().Delete(&system)
	s.db.Unscoped().Delete(&group)
}

func (s *ModelsTestSuite) TestGetBlueButtonID() {
	assert := s.Assert()
	cclfBeneficiary := CCLFBeneficiary{HICN: "HASH_ME", MBI: "NOTHING"}
	bbc := testUtils.BlueButtonClient{}

	bbc.On("GetBlueButtonIdentifier", client.HashHICN(cclfBeneficiary.HICN)).Return(bbc.GetData("Patient", "BB_VALUE"))
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

	// A record with the same HICN value exists.  Grab that value.
	cclfFile := CCLFFile{
		Name:            "HASHTEST",
		CCLFNum:         8,
		ACOCMSID:        "12345",
		PerformanceYear: 2019,
		Timestamp:       time.Now(),
	}
	db.Save(&cclfFile)
	defer db.Unscoped().Delete(&cclfFile)
	cclfBeneficiary.FileID = cclfFile.ID
	// Save a blank one, this shouldn't affect pulling a val from the DB later
	cclfBeneficiary.BlueButtonID = ""
	err = db.Create(&cclfBeneficiary).Error
	defer db.Unscoped().Delete(&cclfBeneficiary)
	assert.Nil(err)
	cclfBeneficiary.ID = 0
	cclfBeneficiary.BlueButtonID = "DB_VALUE"
	err = db.Create(&cclfBeneficiary).Error
	defer db.Unscoped().Delete(&cclfBeneficiary)

	assert.Nil(err)
	newCCLFBeneficiary := CCLFBeneficiary{HICN: cclfBeneficiary.HICN, MBI: "NOT_AN_MBI"}
	newBBID, err := newCCLFBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("DB_VALUE", newBBID)
	// Should be making only a single call to BB for all 3 attempts.
	bbc.AssertNumberOfCalls(s.T(), "GetBlueButtonIdentifier", 1)
}
