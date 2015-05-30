package main

import (
	"bufio"
	"io"
	"os"
)

var (
	// reassignable for testing purposes
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

// readMsg reads and returns a NUL-terminated string from an io.Reader.
func readMsg(r io.Reader) (string, error) {
	msg, err := bufio.NewReader(r).ReadString('\000')
	if len(msg) > 0 {
		msg = msg[:len(msg)-1]
	}
	return msg, err
}
