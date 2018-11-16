package servicemux

import (
	"net"
	"os"
	"testing"

	"github.com/soheilhy/cmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServiceMuxTestSuite struct {
	suite.Suite
}

func (s *ServiceMuxTestSuite) TestAccept() {
	// ln, _ := net.Listen("tcp", ":1234")
	// kln := tcpKeepAliveListener{ln.(*net.TCPListener)}

	// c, err := kln.Accept()

	// assert.Nil(s.T(), err)
	// assert.NotNil(s.T(), c)
	// assert.IsType(s.T(), net.TCPConn{}, c)

	// ln.Close()
	// kln.Close()
	// c.Close()
}

func (s *ServiceMuxTestSuite) TestURLPrefixMatcher() {
	ln, _ := net.Listen("tcp", ":1234")
	cm := cmux.New(ln)
	m := URLPrefixMatcher("test")
	ml := cm.Match(m)
	assert.NotNil(s.T(), ml)
}

func (s *ServiceMuxTestSuite) TestNew() {
	sm := New(":1234")

	assert.NotNil(s.T(), sm)
	assert.Equal(s.T(), ":1234", sm.Addr)
	assert.NotNil(s.T(), sm.Listener)
	assert.IsType(s.T(), tcpKeepAliveListener{}, sm.Listener)
	assert.Empty(s.T(), sm.Servers)
}

func (s *ServiceMuxTestSuite) TestAddServer() {

}

func (s *ServiceMuxTestSuite) TestServe_NoCert() {
	origTLSCert := os.Getenv("BCDA_TLS_CERT")
	origTLSKey := os.Getenv("BCDA_TLS_KEY")
	origHTTPOnly := os.Getenv("HTTP_ONLY")

	defer func() {
		os.Setenv("BCDA_TLS_CERT", origTLSCert)
		os.Setenv("BCDA_TLS_KEY", origTLSKey)
		os.Setenv("HTTP_ONLY", origHTTPOnly)
	}()

	os.Setenv("BCDA_TLS_CERT", "")
	os.Setenv("BCDA_TLS_KEY", "test.key")
	os.Setenv("HTTP_ONLY", "")

	sm := &ServiceMux{}
	assert.Panics(s.T(), sm.Serve)
}

func (s *ServiceMuxTestSuite) TestServe_NoKey() {
	origTLSCert := os.Getenv("BCDA_TLS_CERT")
	origTLSKey := os.Getenv("BCDA_TLS_KEY")
	origHTTPOnly := os.Getenv("HTTP_ONLY")

	defer func() {
		os.Setenv("BCDA_TLS_CERT", origTLSCert)
		os.Setenv("BCDA_TLS_KEY", origTLSKey)
		os.Setenv("HTTP_ONLY", origHTTPOnly)
	}()

	os.Setenv("BCDA_TLS_CERT", "test.crt")
	os.Setenv("BCDA_TLS_KEY", "")
	os.Setenv("HTTP_ONLY", "")

	sm := &ServiceMux{}
	assert.Panics(s.T(), sm.Serve)
}

func (s *ServiceMuxTestSuite) TestServe_serveHTTPS() {
	// origTLSCert := os.Getenv("BCDA_TLS_CERT")
	// origTLSKey := os.Getenv("BCDA_TLS_KEY")

	// defer func() {
	// 	os.Setenv("BCDA_TLS_CERT", origTLSCert)
	// 	os.Setenv("BCDA_TLS_KEY", origTLSKey)
	// }()

	// os.Setenv("BCDA_TLS_CERT", "test")
	// os.Setenv("BCDA_TLS_KEY", "test")

	// sm := &ServiceMux{}
	// sm.Serve()

	// assert.NotNil(s.T(), sm.Listener)
}

// func (s *ServiceMuxTestSuite) TestServe_serveHTTP() {
// 	origTLSCert := os.Getenv("BCDA_TLS_CERT")
// 	origTLSKey := os.Getenv("BCDA_TLS_KEY")
// 	origHTTPOnly := os.Getenv("HTTP_ONLY")

// 	defer func() {
// 		os.Setenv("BCDA_TLS_CERT", origTLSCert)
// 		os.Setenv("BCDA_TLS_KEY", origTLSKey)
// 		os.Setenv("HTTP_ONLY", origHTTPOnly)
// 	}()

// 	os.Setenv("BCDA_TLS_CERT", "")
// 	os.Setenv("BCDA_TLS_KEY", "")
// 	os.Setenv("HTTP_ONLY", "true")

// 	sm := New(":2345")
// 	sm.Serve()

// 	assert.NotNil(s.T(), sm.Listener)

// 	sm.Listener.Close()
// }

func TestServiceMuxTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceMuxTestSuite))
}
