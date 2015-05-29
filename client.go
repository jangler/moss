package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// sendCommand sends a command to the server, then reads and prints output.
func sendCommand(args ...string) {
	// connect to server
	conn, err := net.Dial("tcp", addrFlag)
	if err != nil {
		die(args[0] + ": server not running")
	}
	defer conn.Close()

	// send message
	fmt.Fprint(conn, strings.Join(args, " ")+"\000")

	// read and print response
	msg, _ := bufio.NewReader(conn).ReadString('\000')
	msg = msg[:len(msg)-1]
	if msg != "" {
		if msg[0] == '\033' { // signals an error message
			fmt.Fprint(os.Stderr, msg[1:])
			os.Exit(1)
		} else {
			fmt.Print(msg)
		}
	}
}
