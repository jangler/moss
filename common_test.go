package main

import (
	"bytes"
	"testing"
)

const testAddr = ":1234"

// ioCmp is a helper function for testing i/o functions.
func ioCmp(t *testing.T, name string, got, status int, out, err string,
	outbuf, errbuf *bytes.Buffer) {
	if want := status; want != got {
		t.Errorf("%s: got %#v; want %#v", name, got, want)
	}
	if want, got := out, outbuf.String(); want != got {
		t.Errorf("%s: printed %#v to stdout; want %#v", name, got, want)
	}
	if want, got := err, errbuf.String(); want != got {
		t.Errorf("%s: printed %#v to stderr; want %#v", name, got, want)
	}
	outbuf.Reset()
	errbuf.Reset()
}

func TestReadMsg(t *testing.T) {
	// read empty message
	got, err := readMsg(bytes.NewBufferString("\000"))
	if want := ""; want != got {
		t.Errorf("readMsg: got %#v; want %#v", got, want)
	}
	if err != nil {
		t.Errorf("readMsg: unexpected error: %v", err)
	}

	// read non-empty message
	got, err = readMsg(bytes.NewBufferString("Hello, world!\000"))
	if want := "Hello, world!"; want != got {
		t.Errorf("readMsg: got %#v; want %#v", got, want)
	}
	if err != nil {
		t.Errorf("readMsg: unexpected error: %v", err)
	}

	// read no message; encounter error
	got, err = readMsg(bytes.NewBufferString(""))
	if want := ""; want != got {
		t.Errorf("readMsg: got %#v; want %#v", got, want)
	}
	if err == nil {
		t.Errorf("readMsg: expected error")
	}
}
