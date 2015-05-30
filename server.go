package main

import (
	"container/list"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

const (
	statePause uint8 = iota
	statePlay
	stateStop
)

// Assoc contains a regexp and a command associated with it.
type Assoc struct {
	Regexp *regexp.Regexp
	Cmd    []string
}

var (
	curCmd  *exec.Cmd
	curElem *list.Element
	queue   *list.List
	wd      string
	state   = stateStop
	unlock  = make(chan int) // used as mutex
	assocs  = make(map[string]Assoc)
)

// checkIndices checks indices and err. in error cases, an error message is
// written to w and the function returns false.
func checkIndices(cmd string, indices []int, err error, w io.Writer,
	length int) bool {
	if err != nil {
		fmt.Fprintf(w, "\033%s: invalid index: %v\n", cmd, err)
		return false
	}
	for _, index := range indices {
		if index < 1 || int(index) > length {
			fmt.Fprintf(w, "\033%s: index out of bounds: %d\n", cmd, index)
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

// ls writes the contents of l, whose elements must have string values, to w.
// one line is written for each element of l.
func ls(l *list.List, w io.Writer) {
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Fprintln(w, e.Value.(string))
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
			arg := curElem.Value.(string)
			cmd := []string{"/bin/sh", "-c"}
			for _, assoc := range assocs {
				if assoc.Regexp.MatchString(arg) {
					cmd = assoc.Cmd
					break
				}
			}
			curCmd = exec.Command(cmd[0], append(cmd[1:], arg)...)
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
			curElem = queue.PushFront("") // dummy element
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
			if curElem.Value.(string) == "" { // dummy element
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
func writeStatus(w io.Writer, s uint8, args []string) {
	switch s {
	case statePause:
		fmt.Fprint(w, "paused")
	case statePlay:
		fmt.Fprint(w, "playing")
	case stateStop:
		fmt.Fprint(w, "stopped")
	}
	if args != nil {
		fmt.Fprint(w, ": "+strings.Join(args, " "))
	}
	fmt.Fprintln(w)
}

// handleConn handles a message from a connection and returns true if and only
// if the server should continue to listen.
func handleConn(conn net.Conn) bool {
	<-unlock
	defer func() {
		go func() { unlock <- 1 }()
		fmt.Fprint(conn, "\000")
		conn.Close()
	}()

	msg, _ := readMsg(conn)

	// assemble args
	args := strings.Split(msg, " ")
	for i := 0; i < len(args)-1; i++ {
		if strings.HasSuffix(args[i], `\`) {
			post := args[i+2:]
			args = append(args[:i], args[i][:len(args[i])-1]+" "+args[i+1])
			args = append(args, post...)
			i--
		}
	}

	// shorten file path arguments
	for i, arg := range args {
		if _, err := os.Stat(arg); err == nil {
			if rel, err := filepath.Rel(wd, arg); err == nil {
				args[i] = rel
			}
		}
	}

	// process message
	switch args[0] {
	case "add":
		if len(args) >= 2 {
			for _, arg := range args[1:] {
				queue.PushBack(arg)
			}
		} else {
			fmt.Fprintln(conn, "\033add: not enough arguments")
		}
	case "assoc":
		if len(args) < 3 {
			fmt.Fprintln(conn, "\033assoc: not enough arguments")
			break
		}

		// compile regexp
		re, err := regexp.Compile(args[1])
		if err != nil {
			fmt.Fprintf(conn, "\033assoc: bad regexp: %s\n", args[1])
			break
		}

		// add association
		assocs[args[1]] = Assoc{re, args[2:]}
	case "del":
		if len(args) < 2 {
			fmt.Fprintln(conn, "\033del: not enough arguments")
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
			fmt.Fprintln(conn, "\033insert: not enough arguments")
		}
	case "kill":
		if len(args) == 1 {
			stop()
			return false
		} else {
			fmt.Fprintln(conn, "\033kill: too many arguments")
		}
	case "ls":
		if len(args) == 1 {
			ls(queue, conn)
		} else {
			fmt.Fprintln(conn, "\033ls: too many arguments")
		}
	case "lsassoc":
		if len(args) == 1 {
			for k, v := range assocs {
				fmt.Fprintf(conn, "%s\t%s\n", k, strings.Join(v.Cmd, " "))
			}
		} else {
			fmt.Fprintln(conn, "\033lsassoc: too many arguments")
		}
	case "mv":
		if len(args) < 3 {
			fmt.Fprintln(conn, "\033mv: not enough arguments")
			break
		} else if len(args) > 3 {
			fmt.Fprintln(conn, "\033mv: too many arguments")
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
			fmt.Fprintln(conn, "\033next: too many arguments")
		}
	case "pause":
		if len(args) == 1 {
			pause()
		} else {
			fmt.Fprintln(conn, "\033pause: too many arguments")
		}
	case "play":
		switch len(args) {
		case 1:
			play()
		case 2:
			// parse index
			index, err := strconv.ParseInt(args[1], 10, 0)
			if err != nil {
				fmt.Fprintf(conn, "\033play: invalid index: %s\n", args[1])
				break
			}
			if index < 1 || int(index) > queue.Len() {
				fmt.Fprintf(conn, "\033play: index out of bounds: %d\n", index)
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
			fmt.Fprintln(conn, "\033play: too many arguments")
		}
	case "prev":
		if len(args) == 1 {
			prev()
		} else {
			fmt.Fprintln(conn, "\033prev: too many arguments")
		}
	case "status":
		if len(args) == 1 {
			if curCmd != nil {
				writeStatus(conn, state, curCmd.Args)
			} else {
				writeStatus(conn, state, nil)
			}
		} else {
			fmt.Fprintln(conn, "\033status: too many arguments")
		}
	case "stop":
		if len(args) == 1 {
			stop()
		} else {
			fmt.Fprintln(conn, "\033stop: too many arguments")
		}
	case "toggle":
		if len(args) > 1 {
			fmt.Fprintln(conn, "\033toggle: too many arguments")
			break
		}

		if state == statePlay {
			pause()
		} else {
			play()
		}
	case "unassoc":
		if len(args) < 2 {
			fmt.Fprintln(conn, "\033unassoc: not enough arguments")
			break
		} else if len(args) > 2 {
			fmt.Fprintln(conn, "\033unassoc: too many arguments")
			break
		}

		// remove association
		delete(assocs, args[1])
	default:
		fmt.Fprintf(conn, "\033%s: unknown command\n", args[0])
	}
	return true
}

// startServer starts the server and returns a channel on which the exit
// status of the server program is sent when finished.
func startServer(addr string) <-chan int {
	c, ready := make(chan int), make(chan int)

	go func() {
		// init server
		wd, _ = os.Getwd()
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
