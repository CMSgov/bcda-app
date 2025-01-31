package main

import (
	"bytes"
	"log"
	"testing"
)

func TestMain(t *testing.T) {
	// Capture the output of the main function
	var buf bytes.Buffer
	log.SetOutput(&buf)
	main()
	output := buf.String()

	expectedOutput := "Hello, World!\n"
	if output != expectedOutput {
		t.Errorf("Expected %q but got %q", expectedOutput, output)
	}
}

func TestDummyFunction(t *testing.T) {
	// Capture the output of the dummyFunction
	var buf bytes.Buffer
	log.SetOutput(&buf)
	dummyFunction()
	output := buf.String()

	expectedOutput := "This is a dummy function.\n"
	if output != expectedOutput {
		t.Errorf("Expected %q but got %q", expectedOutput, output)
	}
}
