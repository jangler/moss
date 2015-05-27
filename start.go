package main

import (
	"flag"
	"net"
	"os"
)

const port = "7781"

// handleConn handles a message from a connection and returns true if and only
// if the server should continue to listen.
func handleConn(conn net.Conn) bool {
	// read message from conn
	p := make([]byte, 1024)
	n, _ := conn.Read(p)
	msg := string(p[:n])
	defer conn.Close()

	// process message
	if msg == "kill" {
		return false
	}
	return true
}

func startFunc() {
	// exit on bad usage
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}

	// create tcp server
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		die("start: server already running")
	}
	defer ln.Close()

	// listen for connections
	for {
		conn, _ := ln.Accept()
		if !handleConn(conn) {
			break
		}
	}
}

func init() {
	commands["start"] = &Command{
		Usage: "start",
		Help: `
Start the mq server in the foreground.`,
		Func: startFunc,
	}
}
