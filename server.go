package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	statePause uint8 = iota
	statePlay
	stateStop
)

var (
	curCmd  *exec.Cmd
	curElem *list.Element
	queue   *list.List
	state   = stateStop
)

// del removes each element from l whose 1-based index is in indices.
func del(l *list.List, indices []int) {
	i := 1
	for e := l.Front(); e != nil; i++ {
		next := e.Next()
		for _, index := range indices {
			if index == i {
				l.Remove(e)
				break
			}
		}
		e = next
	}
}

// ls writes the contents of l, whose elements must have []string values, to w.
// one line is written for each element of l.
func ls(l *list.List, w io.Writer) {
	for e := l.Front(); e != nil; e = e.Next() {
		w.Write([]byte(strings.Join(e.Value.([]string), " ") + "\n"))
	}
}

// parseInts converts each string in args to an int and returns a []int of the
// results, or returns an error if an index cannot be converted.
func parseInts(args []string) ([]int, error) {
	ints := make([]int, len(args))
	for i, arg := range args {
		v, err := strconv.ParseInt(arg, 10, 0)
		if err != nil {
			return nil, fmt.Errorf("%s", arg)
		}
		ints[i] = int(v)
	}
	return ints, nil
}

// pause suspends the current command.
func pause() {
	if state == statePlay && curCmd != nil {
		curCmd.Process.Signal(syscall.SIGSTOP)
		state = statePause
	}
}

// play resumes the current command if paused, or starts a new command
// otherwise.
func play() {
	switch state {
	case statePause:
		if curCmd != nil {
			curCmd.Process.Signal(syscall.SIGCONT)
			state = statePlay
		}
	case stateStop:
		if curElem == nil {
			curElem = queue.Front()
		}
		if curElem != nil {
			args := curElem.Value.([]string)
			curCmd = exec.Command(args[0], args[1:]...)
			curCmd.Start()
			state = statePlay
			go waitCmd()
		}
	}
}

// waitCmd waits for cmd to finish, then runs the next command in the queue.
func waitCmd() {
	curCmd.Wait()
	if state == statePlay { // data race
		if curElem != nil {
			curElem = curElem.Next()
		}
		state = stateStop
		if curElem != nil {
			play()
		}
	}
}

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
		} else {
			conn.Write([]byte("\033add: not enough arguments\n"))
		}
	case "del":
		if len(args) < 2 {
			conn.Write([]byte("\033del: not enough arguments\n"))
			break
		}

		// parse indices
		indices, err := parseInts(args[1:])
		if err != nil {
			conn.Write([]byte(
				fmt.Sprintf("\033del: invalid index: %v\n", err)))
			break
		}
		for _, index := range indices {
			if index < 1 || int(index) > queue.Len() {
				conn.Write([]byte(
					fmt.Sprintf("\033del: index out of bounds: %s\n", index)))
				return true
			}
		}

		// TODO: handle case where current command is deleted

		// delete elements
		del(queue, indices)
	case "kill":
		if len(args) == 1 {
			return false
		} else {
			conn.Write([]byte("\033kill: too many arguments\n"))
		}
	case "ls":
		if len(args) == 1 {
			ls(queue, conn)
		} else {
			conn.Write([]byte("\033ls: too many arguments\n"))
		}
	case "pause":
		if len(args) == 1 {
			pause()
		} else {
			conn.Write([]byte("\033pause: too many arguments\n"))
		}
	case "play":
		if len(args) == 1 {
			play()
		} else {
			conn.Write([]byte("\033play: too many arguments\n"))
		}
	case "status":
		if len(args) > 1 {
			conn.Write([]byte("\033status: too many arguments\n"))
			break
		}

		switch state {
		case statePause:
			conn.Write([]byte("paused"))
		case statePlay:
			conn.Write([]byte("playing"))
		case stateStop:
			conn.Write([]byte("stopped"))
		}
		if curElem != nil {
			conn.Write([]byte(": " +
				strings.Join(curElem.Value.([]string), " ")))
		}
		conn.Write([]byte{'\n'})
	case "stop":
		if len(args) > 1 {
			conn.Write([]byte("\033stop: too many arguments\n"))
			break
		}

		if curCmd != nil {
			curCmd.Process.Kill()
			curCmd = nil
		}
		state = stateStop
	case "toggle":
		if len(args) > 1 {
			conn.Write([]byte("\033toggle: too many arguments\n"))
			break
		}

		if state == statePlay {
			pause()
		} else {
			play()
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
