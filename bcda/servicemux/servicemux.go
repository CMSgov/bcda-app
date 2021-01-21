package servicemux

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

    configuration "github.com/CMSgov/bcda-app/config"

	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
)

var keepAliveInterval int = 60

func init() {
	interval := configuration.GetEnv("SERVICE_MUX_KEEP_ALIVE_INTERVAL")
	if interval != "" {
		interval, err := strconv.Atoi(interval)
		if err != nil {
			panic(err)
		}
		keepAliveInterval = interval
	}
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}

	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = tc.SetKeepAlivePeriod(time.Duration(keepAliveInterval) * time.Second)
	if err != nil {
		return nil, err
	}

	return tc, nil
}

func URLPrefixMatcher(prefix string) cmux.Matcher {
	return func(r io.Reader) bool {
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return false
		}
		return strings.HasPrefix(req.URL.Path, prefix)
	}
}

type ServiceMux struct {
	Addr      string
	Listener  net.Listener
	Servers   []map[*http.Server]string
	TLSConfig tls.Config
}

func New(addr string) *ServiceMux {
	s := &ServiceMux{
		Addr: addr,
	}

	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}

	s.Listener = tcpKeepAliveListener{ln.(*net.TCPListener)}
	return s
}

func (sm *ServiceMux) AddServer(s *http.Server, m string) {
	var server = make(map[*http.Server]string)
	server[s] = m
	sm.Servers = append(sm.Servers, server)
}

func (sm *ServiceMux) Serve() {
	tlsCertPath := configuration.GetEnv("BCDA_TLS_CERT")
	tlsKeyPath := configuration.GetEnv("BCDA_TLS_KEY")

	// If HTTP_ONLY is not set or has any value except true, assume HTTPS
	if configuration.GetEnv("HTTP_ONLY") == "true" {
		sm.serveHTTP()
	} else if tlsCertPath != "" && tlsKeyPath != "" {
		sm.serveHTTPS(tlsCertPath, tlsKeyPath)
	} else {
		panic("TLS certificate and key paths are required unless HTTP_ONLY is true")
	}
}

func (sm *ServiceMux) serveHTTPS(tlsCertPath, tlsKeyPath string) {
	certificate, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
	if err != nil {
		log.Panic(err)
	}

	sm.TLSConfig = tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
	}

	sm.Listener = tls.NewListener(sm.Listener, &sm.TLSConfig)

	sm.serveHTTP()
}

func (sm *ServiceMux) serveHTTP() {
	m := cmux.New(sm.Listener)

	for _, server := range sm.Servers {
		for srv, path := range server {
			var match net.Listener

			if path == "" {
				match = m.Match(cmux.Any())
			} else {
				match = m.Match(URLPrefixMatcher(path))
			}

			srv.TLSConfig = &sm.TLSConfig

			//nolint
			go srv.Serve(match)
		}
	}

	err := m.Serve()
	if err != nil {
		panic(err)
	}
}

func (sm *ServiceMux) Close() {
	err := sm.Listener.Close()
	if err != nil {
		log.Panic(err)
	}
}

func IsHTTPS(r *http.Request) bool {
	srvCtxKey := r.Context().Value(http.ServerContextKey)
	if srvCtxKey == nil {
		return false
	}
	srv := srvCtxKey.(*http.Server)
	return srv.TLSConfig.Certificates != nil
}
