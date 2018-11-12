package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
)

var keepAliveInterval int = 30

func init() {
	interval := os.Getenv("SERVICE_MUX_KEEP_ALIVE_INTERVAL")
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
	Addr     string
	Listener net.Listener
	Servers  []map[*http.Server]string
}

func NewServiceMux(addr string) *ServiceMux {
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
	tlsCertPath := os.Getenv("BCDA_TLS_CERT")
	tlsKeyPath := os.Getenv("BCDA_TLS_KEY")

	if tlsCertPath != "" && tlsKeyPath != "" {
		sm.serveHTTPS(tlsCertPath, tlsKeyPath)
	} else if tlsCertPath != "" {
		panic("TLS certificate path provided, but no key")
	} else if tlsKeyPath != "" {
		panic("TLS key path provided, but no certificate")
	} else {
		log.Warn("TLS not enabled")
		sm.serveHTTP()
	}
}

func (sm *ServiceMux) serveHTTPS(tlsCertPath, tlsKeyPath string) {
	certificate, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
	if err != nil {
		log.Panic(err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
	}

	sm.Listener = tls.NewListener(sm.Listener, config)

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

			//nolint
			go srv.Serve(match)
		}
	}

	err := m.Serve()
	if err != nil {
		panic(err)
	}
}
