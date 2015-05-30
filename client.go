package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// printMsg prints the message to standard output or standard error, depending
// on the message contents, and returns the exit status of the client program.
func printMsg(msg string) int {
	if msg != "" {
		if msg[0] == '\033' { // signals an error message
			fmt.Fprint(stderr, msg[1:])
			return 1
		} else {
			fmt.Fprint(stdout, msg)
		}
	}
	return 0
}

// sendCommand sends a command to the server, then reads and prints output.
// the return value is the exit status of the client program.
func sendCommand(addr string, args ...string) int {
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer conn.Close()

	// canonicalize file path arguments
	for i, arg := range args {
		if _, err := os.Stat(arg); err == nil {
			if arg, err := filepath.Abs(arg); err == nil {
				args[i] = filepath.Clean(arg)
			}
		}
	}

	// send message
	for i, arg := range args {
		args[i] = strings.Replace(arg, " ", `\ `, -1) // escape spaces
	}
	fmt.Fprintf(conn, "%s\000", strings.Join(args, " "))

	// handle response
	msg, err := readMsg(conn)
	if err != nil {
		fmt.Fprintln(stderr, err)
	}
	return printMsg(msg)
}
