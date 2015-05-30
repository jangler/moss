package main

import (
	"bytes"
	"flag"
	"os"
	"testing"
)

func TestUsage(t *testing.T) {
	// redirect stderr to buffer
	buf := &bytes.Buffer{}
	stderr = buf
	flag.CommandLine.SetOutput(stderr)

	// not much to test here--just make sure it prints something
	usage()
	if buf.String() == "" {
		t.Errorf("usage: printed empty string")
	}

	// reset stderr to default
	stderr = os.Stderr
	flag.CommandLine.SetOutput(stderr)
}
