package main

import (
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
	unlock  = make(chan int) // used as mutex
)

// checkIndices checks indices and err. in error cases, an error message is
// written to w and the function returns false.
func checkIndices(cmd string, indices []int, err error, w io.Writer,
	length int) bool {
	if err != nil {
		w.Write([]byte(
			fmt.Sprintf("\033%s: invalid index: %v\n", cmd, err)))
		return false
	}
	for _, index := range indices {
		if index < 1 || int(index) > length {
			w.Write([]byte(
				fmt.Sprintf("\033%s: index out of bounds: %d\n", cmd, index)))
			return false
		}
	}
	return true
}

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

// getIndex returns element i of list l. i must be within bounds.
func getIndex(l *list.List, i int) *list.Element {
	e := l.Front()
	for i > 1 {
		e = e.Next()
		i--
	}
	return e
}

// ls writes the contents of l, whose elements must have []string values, to w.
// one line is written for each element of l.
func ls(l *list.List, w io.Writer) {
	for e := l.Front(); e != nil; e = e.Next() {
		w.Write([]byte(strings.Join(e.Value.([]string), " ") + "\n"))
	}
}

// next steps forward in the command list.
func next() {
	if curElem != nil {
		curElem = curElem.Next()
	}
	stepState()
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
			go waitCmd(curCmd)
		}
	}
}

// prev steps backward in the command list.
func prev() {
	if curElem != nil {
		curElem = curElem.Prev()
	}
	stepState()
}

// stepState adjusts the state after a change in the current command.
func stepState() {
	if state == statePlay && curElem != nil && curCmd != nil {
		// kind of ugly, but this all makes sense if you consider waitCmd()
		curElem = curElem.Prev()
		if curElem == nil {
			curElem = queue.PushFront([]string{}) // dummy element
		}
		curCmd.Process.Kill()
	} else {
		stop()
	}
}

// stop kills the current command.
func stop() {
	state = stateStop
	if curCmd != nil {
		if curCmd.Process != nil {
			curCmd.Process.Kill()
		}
		curCmd = nil
	}
}

// waitCmd waits for cmd to finish, then runs the next command in the queue.
func waitCmd(cmd *exec.Cmd) {
	cmd.Wait()
	<-unlock
	if state == statePlay {
		if curElem != nil {
			next := curElem.Next()
			if len(curElem.Value.([]string)) == 0 { // dummy element
				queue.Remove(curElem)
			}
			curElem = next
		}
		state = stateStop
		if curElem != nil {
			play()
		}
	}
	unlock <- 1
}

// writeStatus writes a status message to w depending on the state s and
// current queue element e.
func writeStatus(w io.Writer, s uint8, e *list.Element) {
	switch s {
	case statePause:
		w.Write([]byte("paused"))
	case statePlay:
		w.Write([]byte("playing"))
	case stateStop:
		w.Write([]byte("stopped"))
	}
	if e != nil {
		w.Write([]byte(": " + strings.Join(e.Value.([]string), " ")))
	}
	w.Write([]byte{'\n'})
}

// handleConn handles a message from a connection and returns true if and only
// if the server should continue to listen.
func handleConn(conn net.Conn) bool {
	<-unlock
	defer func() {
		go func() { unlock <- 1 }()
		conn.Write([]byte("\000"))
		conn.Close()
	}()

	msg, _ := readMsg(conn)

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
		if !checkIndices("del", indices, err, conn, queue.Len()) {
			break
		}

		// delete elements
		del(queue, indices)
	case "insert":
		if len(args) >= 2 {
			length := queue.Len()
			if curElem != nil {
				queue.InsertAfter(args[1:], curElem)
			}
			if queue.Len() == length { // insertion failed; insert at front
				queue.PushFront(args[1:])
			}
		} else {
			conn.Write([]byte("\033insert: not enough arguments\n"))
		}
	case "kill":
		if len(args) == 1 {
			stop()
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
	case "mv":
		if len(args) < 3 {
			conn.Write([]byte("\033mv: not enough arguments\n"))
			break
		} else if len(args) > 3 {
			conn.Write([]byte("\033mv: too many arguments\n"))
			break
		}

		// parse indices
		indices, err := parseInts(args[1:])
		if !checkIndices("mv", indices, err, conn, queue.Len()) {
			break
		}

		// get source and target elements
		e1, e2 := getIndex(queue, indices[0]), getIndex(queue, indices[1])

		// move source element to target
		if indices[0] < indices[1] {
			queue.MoveAfter(e1, e2)
		} else {
			queue.MoveBefore(e1, e2)
		}
	case "next":
		if len(args) == 1 {
			next()
		} else {
			conn.Write([]byte("\033next: too many arguments\n"))
		}
	case "pause":
		if len(args) == 1 {
			pause()
		} else {
			conn.Write([]byte("\033pause: too many arguments\n"))
		}
	case "play":
		switch len(args) {
		case 1:
			play()
		case 2:
			// parse index
			index, err := strconv.ParseInt(args[1], 10, 0)
			if err != nil {
				conn.Write([]byte(
					fmt.Sprintf("\033play: invalid index: %s\n", args[1])))
				break
			}
			if index < 1 || int(index) > queue.Len() {
				conn.Write([]byte(
					fmt.Sprintf("\033play: index out of bounds: %d\n", index)))
				break
			}

			// get element
			curElem = getIndex(queue, int(index))

			// start command
			if curCmd != nil {
				state = statePlay
				stepState()
			} else {
				play()
			}
		default:
			conn.Write([]byte("\033play: too many arguments\n"))
		}
	case "prev":
		if len(args) == 1 {
			prev()
		} else {
			conn.Write([]byte("\033prev: too many arguments\n"))
		}
	case "status":
		if len(args) == 1 {
			writeStatus(conn, state, curElem)
		} else {
			conn.Write([]byte("\033status: too many arguments\n"))
		}
	case "stop":
		if len(args) == 1 {
			stop()
		} else {
			conn.Write([]byte("\033stop: too many arguments\n"))
		}
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

// startServer starts the server and returns a channel on which the exit
// status of the server program is sent when finished.
func startServer(addr string) <-chan int {
	c, ready := make(chan int), make(chan int)

	go func() {
		// init queue, tcp server
		queue = list.New()
		ln, err := net.Listen("tcp", addr)
		close(ready)
		if err != nil {
			fmt.Fprintln(stderr, err)
			go func() {
				c <- 1
				close(c)
			}()
			return
		}
		defer ln.Close()

		go func() { unlock <- 1 }() // start mutex in unlocked state

		// listen for connections
		for {
			conn, _ := ln.Accept()
			if conn != nil && !handleConn(conn) {
				break
			}
		}
		go func() {
			c <- 0
			close(c)
		}()
	}()

	<-ready
	return c
}
