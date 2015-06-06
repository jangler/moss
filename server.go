package main

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
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

// clear removes items from l whose string values are matched by re.
func clear(l *list.List, re *regexp.Regexp) {
	del(l, getIndices(l, []*regexp.Regexp{re})) // this is why it's deprecated
}

// del removes each element from l whose 1-based index is in indices.
func del(l *list.List, indices []int) {
	i := 1
	for e := l.Front(); e != nil; i++ {
		next := e.Next()
		for _, index := range indices {
			if index == i {
				if e == curElem {
					stop()
					curElem = nil
				}
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

// getIndices returns a slice of 1-based indices of items in l that match a
// regular expression in res.
func getIndices(l *list.List, res []*regexp.Regexp) []int {
	m := make(map[int]bool) // use map so that each index is only included once

	// get matches
	for _, re := range res {
		i := 1
		for e := l.Front(); e != nil; e = e.Next() {
			if re.MatchString(e.Value.(string)) {
				m[i] = true
			}
			i++
		}
	}

	// return sorted int slice converted from map
	indices := make([]int, len(m))
	i := 0
	for index := range m {
		indices[i] = index
		i++
	}
	sort.Ints(indices)
	return indices
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

// statusMap returns a map of special sequences to the strings they represent.
func statusMap() map[string]string {
	m := make(map[string]string)

	if curElem != nil {
		m["%t"] = curElem.Value.(string)
		if _, err := os.Stat(m["%t"]); err == nil {
			m["%f"] = filepath.Join(wd, m["%t"])
		} else {
			m["%f"] = m["%t"]
		}
		i := 1
		for e := queue.Front(); e != curElem && e != nil; e = e.Next() {
			i++
		}
		m["%i"] = fmt.Sprintf("%d", i)
	} else {
		m["%t"] = ""
		m["%f"] = ""
		m["%i"] = "0"
	}

	m["%n"] = fmt.Sprintf("%d", queue.Len())

	if curCmd != nil {
		m["%c"] = strings.Join(curCmd.Args[:len(curCmd.Args)-1], " ")
		if curCmd.Process != nil {
			m["%p"] = fmt.Sprintf("%d", curCmd.Process.Pid)
		} else {
			m["%p"] = ""
		}
	} else {
		m["%c"] = ""
		m["%p"] = ""
	}

	switch state {
	case statePause:
		m["%s"] = "paused"
	case statePlay:
		m["%s"] = "playing"
	case stateStop:
		m["%s"] = "stopped"
	}

	return m
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
		} else {
			curCmd = nil
		}
	}
	unlock <- 1
}

// writeStatus writes a status message to w depending on the state s and
// current queue element e.
func writeStatus(w io.Writer, m map[string]string, s string) {
	for k, v := range m {
		s = strings.Replace(s, k, v, -1)
	}
	fmt.Fprintln(w, s)
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
	case "clear":
		if len(args) > 2 {
			fmt.Fprintln(conn, "\033clear: too many arguments")
			break
		}

		// compile regexp
		s := ""
		if len(args) == 2 {
			s = args[1]
		}
		re, err := regexp.Compile(s)
		if err != nil {
			fmt.Fprintf(conn, "\033clear: bad regexp: %s\n", s)
			break
		}

		// delete items
		clear(queue, re)
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
	case "index":
		if len(args) < 2 {
			fmt.Fprintln(conn, "\033index: not enough arguments")
			break
		}

		// compile regexps
		res := make([]*regexp.Regexp, len(args) - 1)
		for i, arg := range args[1:] {
			re, err := regexp.Compile(arg)
			if err != nil {
				fmt.Fprintf(conn, "\033index: bad regexp: %s\n", arg)
				return true
			}
			res[i] = re
		}

		// print indices
		for _, index := range getIndices(queue, res) {
			fmt.Fprintf(conn, "%d\n", index)
		}
	case "insert":
		if len(args) >= 2 {
			e := curElem
			for _, arg := range args[1:] {
				length := queue.Len()
				if e != nil {
					e = queue.InsertAfter(arg, e)
				}
				if queue.Len() == length { // insertion failed
					e = queue.PushFront(arg)
				}
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
				fmt.Fprintf(conn, "%s %s\n", k, strings.Join(v.Cmd, " "))
			}
		} else {
			fmt.Fprintln(conn, "\033lsassoc: too many arguments")
		}
	case "mv":
		if len(args) < 3 {
			fmt.Fprintln(conn, "\033mv: not enough arguments")
			break
		}

		// parse indices
		indices, err := parseInts(args[1:])
		if !checkIndices("mv", indices, err, conn, queue.Len()) {
			break
		}

		// get source and dest elements
		sources := make([]*list.Element, len(indices) - 1)
		for i, srcIndex := range indices[:len(indices) - 1] {
			sources[i] = getIndex(queue, srcIndex)
		}
		dst := getIndex(queue, indices[len(indices) - 1])

		// move first source element
		if indices[0] < indices[len(indices) - 1] {
			queue.MoveAfter(sources[0], dst)
		} else {
			queue.MoveBefore(sources[0], dst)
		}
		dst = sources[0]

		// move other source elements
		for _, src := range sources[1:] {
			queue.MoveAfter(src, dst)
			dst = src
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
			writeStatus(conn, statusMap(), "%s #%i/%n: %c %t")
		} else if len(args) == 2 {
			writeStatus(conn, statusMap(), args[1])
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
		}

		// remove associations
		for _, arg := range args[1:] {
			delete(assocs, arg)
		}
	default:
		fmt.Fprintf(conn, "\033%s: unknown command\n", args[0])
	}
	return true
}

// readRCFile reads the RC file, if it exists, and sends each line to the
// server as a command.
func readRCFile(addr string) {
	curUser, err := user.Current()
	if err != nil {
		return
	}

	f, _ := os.Open(filepath.Join(curUser.HomeDir, ".mossrc"))
	if f == nil {
		f, _ = os.Open(filepath.Join(curUser.HomeDir, ".config", "mossrc"))
	}
	if f == nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		if line, err := r.ReadString('\n'); err == nil {
			sendCommand(addr, strings.Split(line[:len(line)-1], " ")...)
		} else {
			break
		}
	}
}

// startServer starts the server and returns a channel on which the exit
// status of the server program is sent when finished.
func startServer(addr string, rc bool) <-chan int {
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
	if rc {
		go readRCFile(addr)
	}
	return c
}
