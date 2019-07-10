package public

import (
	"bytes"
	"github.com/CMSgov/bcda-app/ssas/okta"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"testing"
)

type OTestSuite struct {
	suite.Suite
}

func (s *OTestSuite) TestGetUserHeaders() {
	trackingId := uuid.NewRandom().String()
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		assert.Contains(s.T(), req.Header.Get("Authorization"), "SSWS")
		assert.Equal(s.T(), req.Header.Get("Content-Type"), "application/json")
		assert.Equal(s.T(), req.Header.Get("Accept"), "application/json")
		return testHttpResponse(200, `[]`)
	})

	o := NewOkta(client)
	_, _ = o.GetUser("nonexistent_user", trackingId)
}

func (s *OTestSuite) TestGetUserSuccess() {
	trackingId := uuid.NewRandom().String()
	expectedUserId := "abc123"
	searchString := "a_user@cms.gov"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(s.T(), req.URL.String(), okta.OktaBaseUrl + "/api/v1/users/?q=" + searchString)
		return testHttpResponse(200, `[{"id":"abc123","status":"ACTIVE","created":"2018-12-05T19:48:17.000Z","activated":"2018-12-05T19:48:17.000Z","statusChanged":"2019-06-04T12:52:49.000Z","lastLogin":"2019-06-06T18:45:35.000Z","lastUpdated":"2019-06-04T12:52:49.000Z","passwordChanged":"2019-06-04T12:52:49.000Z","profile":{"firstName":"Test","lastName":"User","mobilePhone":null,"addressType":"Select Type...","secondEmail":"a_user@cms.gov","login":"a_user@cms.gov","email":"a_user@cms.gov","LOA":"3"},"credentials":{"password":{},"emails":[{"value":"a_user@cms.gov","status":"VERIFIED","type":"PRIMARY"},{"value":"a_user@cms.gov","status":"VERIFIED","type":"SECONDARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123"}}}]`)
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser(searchString, trackingId)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), expectedUserId, foundUserId)
}

func (s *OTestSuite) TestGetUserBadStatusCode() {
	trackingId := uuid.NewRandom().String()
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(404, "")
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser("user_irrelevant", trackingId)
	if err == nil {
		s.FailNow("GetUser() should fail unless status code = 200")
	}
	assert.Contains(s.T(), err.Error(), "status code")
	assert.Equal(s.T(), "", foundUserId)
}

func (s *OTestSuite) TestGetUserNotLOA3() {
	trackingId := uuid.NewRandom().String()
	searchString := "a_user@cms.gov"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(200, `[{"id":"abc123","status":"ACTIVE","created":"2018-12-05T19:48:17.000Z","activated":"2018-12-05T19:48:17.000Z","statusChanged":"2019-06-04T12:52:49.000Z","lastLogin":"2019-06-06T18:45:35.000Z","lastUpdated":"2019-06-04T12:52:49.000Z","passwordChanged":"2019-06-04T12:52:49.000Z","profile":{"firstName":"Test","lastName":"User","mobilePhone":null,"addressType":"Select Type...","secondEmail":"a_user@cms.gov","login":"a_user@cms.gov","email":"a_user@cms.gov"},"credentials":{"password":{},"emails":[{"value":"a_user@cms.gov","status":"VERIFIED","type":"PRIMARY"},{"value":"a_user@cms.gov","status":"VERIFIED","type":"SECONDARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123"}}}]`)
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser(searchString, trackingId)
	if err == nil {
		s.FailNow("GetUser() should fail unless LOA=3")
	}
	assert.Contains(s.T(), err.Error(), "LOA")
	assert.Equal(s.T(), "", foundUserId)
}

func (s *OTestSuite) TestGetUserNotActive() {
	trackingId := uuid.NewRandom().String()
	searchString := "a_user@cms.gov"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(200, `[{"id":"abc123","status":"STAGED","created":"2018-12-05T19:48:17.000Z","activated":"2018-12-05T19:48:17.000Z","statusChanged":"2019-06-04T12:52:49.000Z","lastLogin":"2019-06-06T18:45:35.000Z","lastUpdated":"2019-06-04T12:52:49.000Z","passwordChanged":"2019-06-04T12:52:49.000Z","profile":{"firstName":"Test","lastName":"User","mobilePhone":null,"addressType":"Select Type...","secondEmail":"a_user@cms.gov","login":"a_user@cms.gov","email":"a_user@cms.gov","LOA":"3"},"credentials":{"password":{},"emails":[{"value":"a_user@cms.gov","status":"VERIFIED","type":"PRIMARY"},{"value":"a_user@cms.gov","status":"VERIFIED","type":"SECONDARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123"}}}]`)
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser(searchString, trackingId)
	if err == nil {
		s.FailNow("GetUser() should fail unless status=ACTIVE")
	}
	assert.Contains(s.T(), err.Error(), "active")
	assert.Equal(s.T(), "", foundUserId)
}

func (s *OTestSuite) TestGetUserMultipleUsers() {
	trackingId := uuid.NewRandom().String()
	searchString := "a_user"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(200, `[{"id":"def456","status":"ACTIVE","created":"2019-06-04T13:21:06.000Z","activated":"2019-06-04T13:21:07.000Z","statusChanged":"2019-06-04T13:21:07.000Z","lastLogin":null,"lastUpdated":"2019-06-04T13:21:07.000Z","passwordChanged":"2019-06-04T13:21:07.000Z","profile":{"firstName":"Test2","lastName":"User","mobilePhone":null,"secondEmail":"","login":"bcda_user1","email":"bcda_user1@cms.gov"},"credentials":{"password":{},"emails":[{"value":"bcda_user1@cms.gov","status":"VERIFIED","type":"PRIMARY"},{"value":"","status":"VERIFIED","type":"SECONDARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/def456"}}},{"id":"ghi789","status":"STAGED","created":"2019-06-04T13:22:55.000Z","activated":null,"statusChanged":null,"lastLogin":null,"lastUpdated":"2019-06-04T16:34:21.000Z","passwordChanged":null,"profile":{"firstName":"Test3","lastName":"User","aco_ids":["A0000","A0001"],"mobilePhone":null,"addressType":"Select Type...","secondEmail":null,"login":"bcda_user3@cms.gov","email":"bcda_user3@cms.gov","LOA":"Select Level..."},"credentials":{"emails":[{"value":"bcda_user3@cms.gov","status":"VERIFIED","type":"PRIMARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/ghi789"}}}]`)
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser(searchString, trackingId)
	if err == nil {
		s.FailNow("GetUser() should fail unless a single user matches the search string")
	}
	assert.Contains(s.T(), err.Error(), "multiple")
	assert.Equal(s.T(), "", foundUserId)
}

func (s *OTestSuite) TestGetUserNoUsers() {
	trackingId := uuid.NewRandom().String()
	searchString := "no_match_expected"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(200, `[]`)
	})

	o := NewOkta(client)
	foundUserId, err := o.GetUser(searchString, trackingId)
	if err == nil {
		s.FailNow("GetUser() should fail unless a user matches the search string")
	}
	assert.Contains(s.T(), err.Error(), "not found")
	assert.Equal(s.T(), "", foundUserId)
}

func (s *OTestSuite) TestGetUserFactorSuccess() {
	trackingId := uuid.NewRandom().String()
	userId := "abc123"
	factorType := "SMS"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(s.T(), req.URL.String(), okta.OktaBaseUrl + "/api/v1/users/" + userId + "/factors")
		return testHttpResponse(200, `[{"id":"123abc","factorType":"call","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-06-05T14:13:57.000Z","lastUpdated":"2019-06-05T14:13:57.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123def","factorType":"email","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","profile":{"email":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123ghi","factorType":"sms","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-06-05T14:10:19.000Z","lastUpdated":"2019-06-05T14:10:19.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123jkl","factorType":"token:software:totp","provider":"GOOGLE","vendorName":"GOOGLE","status":"ACTIVE","created":"2018-12-05T20:38:23.000Z","lastUpdated":"2018-12-05T20:38:47.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123mno","factorType":"push","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov","deviceType":"SmartPhone_IPhone","keys":[{"kty":"PKIX","use":"sig","kid":"default","x5c":["MIIBI..."]}],"name":"A User’s iPhone","platform":"IOS","version":"12.1.2"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123pqr","factorType":"token:software:totp","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}}]`)
	})

	o := NewOkta(client)
	factor, err := o.GetUserFactor(userId, factorType, trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123ghi", factor.Id)
	assert.Equal(s.T(), "sms", factor.Type)
}

func (s *OTestSuite) TestGetUserFactorAllTypes() {
	trackingId := uuid.NewRandom().String()
	userId := "abc123"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		return testHttpResponse(200, `[{"id":"123abc","factorType":"call","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-06-05T14:13:57.000Z","lastUpdated":"2019-06-05T14:13:57.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123def","factorType":"email","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","profile":{"email":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123ghi","factorType":"sms","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-06-05T14:10:19.000Z","lastUpdated":"2019-06-05T14:10:19.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123jkl","factorType":"token:software:totp","provider":"GOOGLE","vendorName":"GOOGLE","status":"ACTIVE","created":"2018-12-05T20:38:23.000Z","lastUpdated":"2018-12-05T20:38:47.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123mno","factorType":"push","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov","deviceType":"SmartPhone_IPhone","keys":[{"kty":"PKIX","use":"sig","kid":"default","x5c":["MIIBI..."]}],"name":"A User’s iPhone","platform":"IOS","version":"12.1.2"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123pqr","factorType":"token:software:totp","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}}]`)
	})
	o := NewOkta(client)

	factor, err := o.GetUserFactor(userId, "Call", trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123abc", factor.Id)
	assert.Equal(s.T(), "call", factor.Type)

	factor, err = o.GetUserFactor(userId, "Email", trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123def", factor.Id)
	assert.Equal(s.T(), "email", factor.Type)

	factor, err = o.GetUserFactor(userId, "Google TOTP", trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123jkl", factor.Id)
	assert.Equal(s.T(), "token:software:totp", factor.Type)
	assert.Equal(s.T(), "GOOGLE", factor.Provider)

	factor, err = o.GetUserFactor(userId, "OKTA TOTP", trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123pqr", factor.Id)
	assert.Equal(s.T(), "token:software:totp", factor.Type)
	assert.Equal(s.T(), "OKTA", factor.Provider)

	factor, err = o.GetUserFactor(userId, "Push", trackingId)
	assert.Nil(s.T(), err)
	if factor == nil {
		s.FailNow("GetUserFactor() should successfully return a factor")
	}
	assert.Equal(s.T(), "123mno", factor.Id)
	assert.Equal(s.T(), "push", factor.Type)
}

func (s *OTestSuite) TestGetUserFactorInactive() {
	trackingId := uuid.NewRandom().String()
	userId := "abc123"
	factorType := "Call"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(s.T(), req.URL.String(), okta.OktaBaseUrl + "/api/v1/users/" + userId + "/factors")
		return testHttpResponse(200, `[{"id":"123abc","factorType":"call","provider":"OKTA","vendorName":"OKTA","status":"PENDING","created":"2019-06-05T14:13:57.000Z","lastUpdated":"2019-06-05T14:13:57.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123abc/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123def","factorType":"email","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","profile":{"email":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123ghi","factorType":"sms","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-06-05T14:10:19.000Z","lastUpdated":"2019-06-05T14:10:19.000Z","profile":{"phoneNumber":"+15555555555"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123ghi/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123jkl","factorType":"token:software:totp","provider":"GOOGLE","vendorName":"GOOGLE","status":"ACTIVE","created":"2018-12-05T20:38:23.000Z","lastUpdated":"2018-12-05T20:38:47.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123mno","factorType":"push","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov","deviceType":"SmartPhone_IPhone","keys":[{"kty":"PKIX","use":"sig","kid":"default","x5c":["MIIBI..."]}],"name":"A User’s iPhone","platform":"IOS","version":"12.1.2"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123pqr","factorType":"token:software:totp","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}}]`)
	})

	o := NewOkta(client)
	factor, err := o.GetUserFactor(userId, factorType, trackingId)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), factor)
}

func (s *OTestSuite) TestGetUserFactorNotFound() {
	trackingId := uuid.NewRandom().String()
	userId := "abc123"
	factorType := "Call"
	client := okta.NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(s.T(), req.URL.String(), okta.OktaBaseUrl + "/api/v1/users/" + userId + "/factors")
		return testHttpResponse(200, `[{"id":"123def","factorType":"email","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","profile":{"email":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123def/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123jkl","factorType":"token:software:totp","provider":"GOOGLE","vendorName":"GOOGLE","status":"ACTIVE","created":"2018-12-05T20:38:23.000Z","lastUpdated":"2018-12-05T20:38:47.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123jkl/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123mno","factorType":"push","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov","deviceType":"SmartPhone_IPhone","keys":[{"kty":"PKIX","use":"sig","kid":"default","x5c":["MIIBI..."]}],"name":"A User’s iPhone","platform":"IOS","version":"12.1.2"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123mno/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}},{"id":"123pqr","factorType":"token:software:totp","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2019-01-03T18:18:52.000Z","lastUpdated":"2019-01-03T18:19:04.000Z","profile":{"credentialId":"a_user@cms.gov"},"_links":{"self":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr","hints":{"allow":["GET","DELETE"]}},"verify":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123/factors/123pqr/verify","hints":{"allow":["POST"]}},"user":{"href":"https://cms-sandbox.oktapreview.com/api/v1/users/abc123","hints":{"allow":["GET"]}}}}]`)
	})

	o := NewOkta(client)
	factor, err := o.GetUserFactor(userId, factorType, trackingId)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), factor)
}

func TestOTestSuite(t *testing.T) {
	suite.Run(t, new(OTestSuite))
}

func testHttpResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
