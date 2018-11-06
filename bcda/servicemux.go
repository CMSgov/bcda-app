package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

	s.Listener = ln
	return s
}

func (sm *ServiceMux) AddServer(s *http.Server, m string) {
	var server = make(map[*http.Server]string)
	server[s] = m
	sm.Servers = append(sm.Servers, server)
}

func (sm *ServiceMux) Serve() {
	m := cmux.New(tcpKeepAliveListener{sm.Listener.(*net.TCPListener)})

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
