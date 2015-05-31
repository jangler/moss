package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

// testPrintMsg is a helper function for testing printMsg.
func testPrintMsg(t *testing.T, msg string, status int, out, err string,
	outbuf, errbuf *bytes.Buffer) {
	ioCmp(t, "printMsg", printMsg(msg), status, out, err, outbuf, errbuf)
}

func TestPrintMsg(t *testing.T) {
	// redirect stdout and stderr to buffers
	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout, stderr = outbuf, errbuf

	// print empty message, normal message, error message
	testPrintMsg(t, "", 0, "", "", outbuf, errbuf)
	testPrintMsg(t, "hello world", 0, "hello world", "", outbuf, errbuf)
	testPrintMsg(t, "\033bad error", 1, "", "bad error", outbuf, errbuf)

	// reset stdout and stderr to defaults
	stdout, stderr = os.Stdout, os.Stderr
}

// testSendCommand is a helper function for testing sendCommand.
func testSendCommand(t *testing.T, addr string, args []string, status int,
	out, err string, outbuf, errbuf *bytes.Buffer) {
	ioCmp(t, "sendCommand", sendCommand(addr, args...), status, out, err,
		outbuf, errbuf)
}

func TestSendCommand(t *testing.T) {
	// redirect stdout and stderr to buffers
	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout, stderr = outbuf, errbuf

	// test connection error
	testSendCommand(t, testAddr, []string{"arg"}, 1,
		"", "dial tcp :1234: connection refused\n", outbuf, errbuf)

	startServer(testAddr, false)

	// test invalid command
	testSendCommand(t, testAddr, []string{"cmd"}, 1,
		"", "cmd: unknown command\n", outbuf, errbuf)

	// test not enough arguments
	cmds := []string{"add", "assoc", "del", "insert", "mv", "unassoc"}
	for _, cmd := range cmds {
		testSendCommand(t, testAddr, []string{cmd}, 1,
			"", fmt.Sprintf("%s: not enough arguments\n", cmd), outbuf, errbuf)
	}

	// test too many arguments (2)
	cmds = []string{"kill", "ls", "lsassoc", "next", "pause", "prev", "status",
		"stop", "toggle"}
	for _, cmd := range cmds {
		testSendCommand(t, testAddr, []string{cmd, "1"}, 1,
			"", fmt.Sprintf("%s: too many arguments\n", cmd), outbuf, errbuf)
	}

	// test too many arguments (3)
	for _, cmd := range []string{"clear", "unassoc"} {
		testSendCommand(t, testAddr, []string{cmd, "1", "2"}, 1,
			"", fmt.Sprintf("%s: too many arguments\n", cmd), outbuf, errbuf)
	}

	// test valid command
	testSendCommand(t, testAddr, []string{"status"}, 0, "stopped\n", "",
		outbuf, errbuf)

	sendCommand(testAddr, "kill")

	// reset stdout and stderr to defaults
	stdout, stderr = os.Stdout, os.Stderr
}
