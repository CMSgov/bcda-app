package public

import (
	"context"
	"github.com/CMSgov/bcda-app/ssas/service"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

type PublicMiddlewareTestSuite struct {
	suite.Suite
	server *httptest.Server
	rr     *httptest.ResponseRecorder
}

func (s *PublicMiddlewareTestSuite) CreateRouter(handler ...func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	router.With(handler...).Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	return router
}

func (s *PublicMiddlewareTestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func (s *PublicMiddlewareTestSuite) TestRequireTokenAuthWithInvalidSignature() {
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"

	testForToken :=
		func (next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := r.Context().Value("token")
				assert.Nil(s.T(), token)
				_, err := readRegData(r)
				assert.NotNil(s.T(), err)
			})
		}
	s.server = httptest.NewServer(s.CreateRouter(parseToken, testForToken))
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	req.Header.Add("Authorization", "Bearer " + badToken)

	resp, err := client.Do(req)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *PublicMiddlewareTestSuite) TestParseTokenEmptyToken() {
	testForToken :=
		func (next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := r.Context().Value("token")
				assert.Nil(s.T(), token)
				_, err := readRegData(r)
				assert.NotNil(s.T(), err)
			})
		}
	s.server = httptest.NewServer(s.CreateRouter(parseToken, testForToken))
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer ")

	_, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *PublicMiddlewareTestSuite) TestParseTokenValidToken() {
	oktaID := "fake_okta_id"
	groupIDs := []string{"T0001", "T0002"}
	testForToken :=
		func (next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ts := r.Context().Value("ts")
				assert.NotNil(s.T(), ts)
				rd, err := readRegData(r)
				if err != nil {
					assert.FailNow(s.T(), err.Error())
				}
				assert.NotNil(s.T(), rd)
				assert.Equal(s.T(), oktaID, rd.OktaID)
				assert.Equal(s.T(), groupIDs, rd.AllowedGroupIDs)
			})
		}
	s.server = httptest.NewServer(s.CreateRouter(parseToken, testForToken))
	client := s.server.Client()

	_, ts, _ := MintRegistrationToken(oktaID, groupIDs)

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	req.Header.Add("Authorization", "Bearer " + ts)

	res, err := client.Do(req)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *PublicMiddlewareTestSuite) TestRequireRegTokenAuthValidToken() {
	s.server = httptest.NewServer(s.CreateRouter(requireRegTokenAuth))

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	handler := requireRegTokenAuth(mockHandler)

	groupIDs := []string{"A0001", "A0002"}
	token, ts, err := MintRegistrationToken("fake_okta_id", groupIDs)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)

	ctx := req.Context()
	ctx = context.WithValue(ctx, "ts", ts)
	req = req.WithContext(ctx)

	handler.ServeHTTP(s.rr, req)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}

func (s *PublicMiddlewareTestSuite) TestRequireRegTokenAuthRevoked() {
	s.server = httptest.NewServer(s.CreateRouter(requireMFATokenAuth))

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	handler := requireMFATokenAuth(mockHandler)

	groupIDs := []string{"A0001", "A0002"}
	token, ts, err := MintRegistrationToken("fake_okta_id", groupIDs)
	assert.Nil(s.T(), err)

	claims := token.Claims.(*service.CommonClaims)
	err = service.TokenBlacklist.BlacklistToken(claims.Id, service.TokenCacheLifetime)
	assert.Nil(s.T(), err)
	assert.True(s.T(), service.TokenBlacklist.IsTokenBlacklisted(claims.Id))

	assert.NotNil(s.T(), token)

	ctx := req.Context()
	ctx = context.WithValue(ctx, "ts", ts)
	req = req.WithContext(ctx)

	handler.ServeHTTP(s.rr, req)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)
}


func (s *PublicMiddlewareTestSuite) TestRequireRegTokenAuthEmptyToken() {
	s.server = httptest.NewServer(s.CreateRouter(requireMFATokenAuth))
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	ctx := context.WithValue(context.Background(), "ts", nil)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *PublicMiddlewareTestSuite) TestRequireMFATokenAuthValidToken() {
	s.server = httptest.NewServer(s.CreateRouter(requireMFATokenAuth))

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	handler := requireMFATokenAuth(mockHandler)
	token, ts, err := MintMFAToken("fake_okta_id")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)

	ctx := req.Context()
	ctx = context.WithValue(ctx, "ts", ts)
	req = req.WithContext(ctx)

	handler.ServeHTTP(s.rr, req)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
}


func (s *PublicMiddlewareTestSuite) TestRequireMFATokenAuthEmptyToken() {
	s.server = httptest.NewServer(s.CreateRouter(requireMFATokenAuth))
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	ctx := context.WithValue(context.Background(), "ts", nil)

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *PublicMiddlewareTestSuite) TestContains() {
	list := []string{"abc", "def", "hij", "hij"}
	assert.True(s.T(), contains(list, "abc"))
	assert.True(s.T(), contains(list, "def"))
	assert.True(s.T(), contains(list, "hij"))
	assert.False(s.T(), contains(list, "lmn"))
}

func TestPublicMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(PublicMiddlewareTestSuite))
}

