package main

import (
	"fmt"
	"net"
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

	// send message
	fmt.Fprint(conn, strings.Join(args, " ")+"\000")

	// handle response
	msg, err := readMsg(conn)
	if err != nil {
		fmt.Fprintln(stderr, err)
	}
	return printMsg(msg)
}
