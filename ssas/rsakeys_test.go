package ssas

import (
	"crypto/rsa"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RSAKeysTestSuite struct {
	suite.Suite
}

func (s *RSAKeysTestSuite) TestReadPrivateKey_EmptyKey() {
	pemData, err := ReadPEMFile("")
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "no such file or directory", "expected os.open error")
	_, err = ReadPrivateKey(pemData)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "empty or nil")
}

func (s *RSAKeysTestSuite) TestReadPrivateKey_BadKeys() {
	assertT := assert.New(s.T())		// is this construction duplicating what s.T() does?
	var tests = []struct {
		path   string
		errMsg string
	}{
		{"SSAS_BAD_BASE64_TEST_PRIVATE_KEY", "decode"},
		{"SSAS_NOT_RSA_TEST_PRIVATE_KEY", "parse RSA"},
		{"SSAS_TOO_SMALL_TEST_PRIVATE_KEY", "insecure key length"},
	}
	for _, test := range tests {
		filePath := os.Getenv(test.path)
		pemData, err := ReadPEMFile(filePath)
		assertT.Nil(err)
		_, err = ReadPrivateKey(pemData)
		assertT.NotNil(err)
		assertT.Contains(err.Error(), test.errMsg)
	}
}

func (s *RSAKeysTestSuite) TestReadPrivateKey_GoodKeys() {
	filePath := os.Getenv("SSAS_GOOD_TEST_PRIVATE_KEY")
	pemData, err := ReadPEMFile(filePath)
	assert.Nil(s.T(), err)
	privateKey, err := ReadPrivateKey(pemData)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &rsa.PrivateKey{}, privateKey)
}

func (s *RSAKeysTestSuite) TestConvertJWKToPEMValid() {
	var jwk1 = `{"alg":"RS256","e":"AQAB","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kid":"wyMwK4A6CL9Qw11uofVeyQ119XyX-xykymkkXygZ5OM","kty":"RSA","use":"enc"}`
	var jwk2 = `{"e":"AAEAAQ","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA"}`

	pem1, err := ConvertJWKToPEM(jwk1)
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), pem1)
	pub1, err := ReadPublicKey(pem1)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), pub1)

	pem2, err := ConvertJWKToPEM(jwk2)
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), pem2)
	pub2, err := ReadPublicKey(pem2)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), pub2)
}

func (s *RSAKeysTestSuite) TestConvertJWKToPEMInvalid() {
	jwkForSig := `{"alg":"RS256","e":"AQAB","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbWAzyqw","kty":"RSA","use":"sig"}`
	jwkECKeyType := `{"kty":"EC","crv":"P-256","x":"MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4","y":"4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM","use":"enc","kid":"1"}`
	jwkCorruptKey := `{"alg":"RS256","e":"AQAB","n":"ok6rvXu95337IxsDXrKzlIqw_I_zPDG8JyEw2CTOtNMoDi1QzpXQVMGj2snNEmvNYaCTmFf51I-EDgeFLLexr40jzBXlg72quV4aw4yiNuxkigW0gMA92OmaT2jMRIdDZM8mVokoxyPfLub2YnXHFq0XuUUgkX_TlutVhgGbyPN0M12teYZtMYo2AUzIRggONhHvnibHP0CPWDjCwSfp3On1Recn4DPxbn3DuGslF2myalmCtkujNcrhHLhwYPP-yZFb8e0XSNTcQvXaQxAqmnWH6NXcOtaeWMQe43PNTAyNinhndgI8ozG3Hz-1NzHssDH_yk6UYFSszhDbW","kty":"RSA","use":"enc"}`

	pem1, err := ConvertJWKToPEM(jwkForSig)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "use type")
	assert.Empty(s.T(), pem1)

	pem2, err := ConvertJWKToPEM(jwkECKeyType)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "key type")
	assert.Empty(s.T(), pem2)

	pem3, err := ConvertJWKToPEM(jwkCorruptKey)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "error in key n value")
	assert.Empty(s.T(), pem3)
}

func TestRSAKeysTestSuite(t *testing.T) {
	suite.Run(t, new(RSAKeysTestSuite))
}
