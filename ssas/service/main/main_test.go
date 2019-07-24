package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/ssas"
)

func TestSSASMain(t *testing.T) {
	var str bytes.Buffer
	ssas.Logger.SetOutput(&str)
	main()
	s := str.String()
	if !strings.Contains(s, "Future home of") {
		t.Errorf("expected log containing 'Future home of'; got %s", s)
	}
}