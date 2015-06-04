package main

import (
	"bytes"
	"flag"
	"os"
	"strings"
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

func TestReadLines(t *testing.T) {
	// test on empty reader
	buf := &bytes.Buffer{}
	if want, got := "", strings.Join(readLines(buf), ", "); want != got {
		t.Errorf("readLines() == %#v; want %#v", got, want)
	}

	// test on non-empty reader
	buf.WriteString("hello\nworld")
	if want, got := "hello, world",
		strings.Join(readLines(buf), ", "); want != got {
		t.Errorf("readLines() == %#v; want %#v", got, want)
	}
}
