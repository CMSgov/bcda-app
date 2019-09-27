package public

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
	"testing"

	"github.com/CMSgov/bcda-app/ssas/service"
)

type PublicTokenTestSuite struct {
	suite.Suite
	server *service.Server
}

func (s *PublicTokenTestSuite) SetupSuite() {
	info := make(map[string][]string)
	info["public"] = []string{"token", "register"}
	s.server = Server()
	err := os.Setenv("DEBUG", "true")
	assert.Nil(s.T(), err)
}

func (s *PublicTokenTestSuite) TestMintMFAToken() {
	token, ts, err := MintMFAToken("my_okta_id")

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *PublicTokenTestSuite) TestMintMFATokenMissingID() {
	token, ts, err := MintMFAToken("")

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(), "", ts)
}

func (s *PublicTokenTestSuite) TestMintRegistrationToken() {
	groupIDs := []string{"A0000", "A0001"}
	token, ts, err := MintRegistrationToken("my_okta_id", groupIDs)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *PublicTokenTestSuite) TestMintRegistrationTokenMissingID() {
	groupIDs := []string{"", ""}
	token, ts, err := MintRegistrationToken("my_okta_id", groupIDs)

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(), "", ts)
}

func (s *PublicTokenTestSuite) TestMintAccessToken() {
	data := `"{\"cms_ids\":[\"T67890\",\"T54321\"]}"`
	t, ts, err := MintAccessToken("2", "0c527d2e-2e8a-4808-b11d-0fa06baf8254", data)

	require.Nil(s.T(), err, )
	assert.NotEmpty(s.T(), ts, "missing token string value")
	assert.NotNil(s.T(), t, "missing token value")

	claims := t.Claims.(*service.CommonClaims)
	assert.NotNil(s.T(), claims.Data, "missing data claim")
	type XData struct {
		IDList []string `json:"cms_ids"`
	}

	var xData XData
	d, err := strconv.Unquote(claims.Data)
	require.Nil(s.T(), err, "couldn't unquote ", d)
	err = json.Unmarshal([]byte(d), &xData)
	require.Nil(s.T(), err, "unexpected error in: ", d)
	require.NotEmpty(s.T(), xData, "no data in data :(")
	assert.Equal(s.T(), 2, len(xData.IDList))
	assert.Equal(s.T(), "T67890", xData.IDList[0])
	assert.Equal(s.T(), "T54321", xData.IDList[1])
}

func (s *PublicTokenTestSuite) TestCheckTokenClaimsMissingType() {
	c := service.CommonClaims{}
	err := checkTokenClaims(&c)
	if err == nil {
		assert.FailNow(s.T(), "must have error with missing token type")
	}
	assert.Contains(s.T(), err.Error(), "missing token type claim")
}

func (s *PublicTokenTestSuite) TestEmpty() {
	groupIDs := []string{"", ""}
	assert.True(s.T(), empty(groupIDs))

	groupIDs = []string{"", "asdf"}
	assert.False(s.T(), empty(groupIDs))
}

func TestPublicTokenTestSuite(t *testing.T) {
	suite.Run(t, new(PublicTokenTestSuite))
}
