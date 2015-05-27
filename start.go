package main

import (
	"container/list"
	"flag"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const port = "7781"

var queue *list.List
var curCmd *exec.Cmd

// handleConn handles a message from a connection and returns true if and only
// if the server should continue to listen.
func handleConn(conn net.Conn) bool {
	// read message from conn
	p := make([]byte, 1024)
	n, _ := conn.Read(p)
	msg := string(p[:n])
	defer conn.Close()

	// process message
	args := strings.Split(msg, " ")
	switch args[0] {
	case "kill":
		return false
	case "play":
		var i, index int64
		if len(args) == 2 {
			index, _ = strconv.ParseInt(args[1], 10, 0)
		}
		var e *list.Element
		for e = queue.Front(); e != nil && i < index-1; e = e.Next() {
			i++
		}
		if e == nil {
			break
		}

		// kill running command
		if curCmd != nil {
			curCmd.Process.Kill()
		}

		args = e.Value.([]string)
		curCmd = exec.Command(args[0], args[1:]...)
		curCmd.Start()
	case "push":
		if len(args) >= 2 {
			queue.PushBack(args[1:])
		}
	case "stop":
		if curCmd != nil {
			curCmd.Process.Kill()
		}
	}
	return true
}

func startFunc() {
	// exit on bad usage
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}

	// init queue, tcp server
	queue = list.New()
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
