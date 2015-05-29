package main

import (
	"bufio"
	"container/list"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

var queue *list.List
var curCmd *exec.Cmd

// handleConn handles a message from a connection and returns true if and only
// if the server should continue to listen.
func handleConn(conn net.Conn) bool {
	defer conn.Close()
	defer conn.Write([]byte("\000"))

	// read message from conn
	msg, _ := bufio.NewReader(conn).ReadString('\000')
	msg = msg[:len(msg)-1]

	// process message
	args := strings.Split(msg, " ")
	switch args[0] {
	case "add":
		if len(args) >= 2 {
			queue.PushBack(args[1:])
		}
	case "del":
		if len(args) < 2 {
			conn.Write([]byte("\033del: not enough arguments\n"))
			break
		}

		// parse indices
		indices := make([]int, len(args)-1)
		for i, arg := range(args[1:]) {
			index, err := strconv.ParseInt(arg, 10, 0)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("\033del: invalid index: %s\n",
					arg)))
				return true
			}
			if index < 1 || int(index) > queue.Len() {
				conn.Write([]byte(
					fmt.Sprintf("\033del: index out of bounds: %s\n", arg)))
				return true
			}
			indices[i] = int(index)
		}

		// delete elements
		i := 1
		for e := queue.Front(); e != nil; i++ {
			next := e.Next()
			for _, index := range indices {
				if index == i {
					queue.Remove(e)
					break
				}
			}
			e = next
		}
	case "kill":
		return false
	case "ls":
		for e := queue.Front(); e != nil; e = e.Next() {
			conn.Write([]byte(strings.Join(e.Value.([]string), " ") + "\n"))
		}
	case "play":
		var i, index int64
		var err error
		if len(args) == 2 {
			index, err = strconv.ParseInt(args[1], 10, 0)
			if err != nil {
				conn.Write([]byte(fmt.Sprintf("\033play: invalid index: %s\n",
					args[1])))
				break
			}
		}
		var e *list.Element
		for e = queue.Front(); e != nil && i < index-1; e = e.Next() {
			i++
		}
		if e == nil {
			conn.Write([]byte(
				fmt.Sprintf("\033play: index out of bounds: %s\n", index)))
			break
		}

		// kill running command
		if curCmd != nil {
			curCmd.Process.Kill()
		}

		args = e.Value.([]string)
		curCmd = exec.Command(args[0], args[1:]...)
		curCmd.Start()
	case "stop":
		if curCmd != nil {
			curCmd.Process.Kill()
		}
	default:
		conn.Write([]byte{'\033'})
		conn.Write([]byte(fmt.Sprintf("%s: unknown command\n", args[0])))
	}
	return true
}

// startServer starts the server.
func startServer() {
	// init queue, tcp server
	queue = list.New()
	ln, err := net.Listen("tcp", addrFlag)
	if err != nil {
		die("-start: server already running")
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
