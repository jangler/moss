package main

import (
	"bytes"
	"container/list"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// stringFromList returns a string representation of l.
func stringFromList(l *list.List) string {
	a := make([]string, l.Len())
	i := 0
	for e := l.Front(); e != nil; e = e.Next() {
		a[i] = fmt.Sprintf("%v", e.Value)
		i++
	}
	return strings.Join(a, ", ")
}

func TestCheckIndices(t *testing.T) {
	// init
	l := list.New()
	for _, v := range []int{1, 2, 3, 4, 5} {
		l.PushBack(v)
	}

	// test success
	buf := &bytes.Buffer{}
	got := checkIndices("t", []int{1, 2, 3, 3, 4, 5}, nil, buf, l.Len())
	if want := true; want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}
	if want, got := "", buf.String(); want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}

	// test err failure
	buf.Reset()
	got = checkIndices("t", []int{}, fmt.Errorf("e"), buf, l.Len())
	if want := false; want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}
	if want, got := "\033t: invalid index: e\n", buf.String(); want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}

	// test bounds failure
	buf.Reset()
	got = checkIndices("t", []int{0}, nil, buf, l.Len())
	if want := false; want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}
	if want, got := "\033t: index out of bounds: 0\n",
		buf.String(); want != got {
		t.Errorf("checkIndices: got %#v; want %#v", got, want)
	}
}

func TestClear(t *testing.T) {
	// init
	l := list.New()
	for _, v := range []string{"alp", "bet", "gam", "del"} {
		l.PushBack(v)
	}

	// clear nothing
	clear(l, regexp.MustCompile("nothing"))
	if want, got := "alp, bet, gam, del", stringFromList(l); want != got {
		t.Errorf("clear: got %#v; want %#v", got, want)
	}

	// clear some
	clear(l, regexp.MustCompile("a"))
	if want, got := "bet, del", stringFromList(l); want != got {
		t.Errorf("clear: got %#v; want %#v", got, want)
	}

	// clear all
	clear(l, regexp.MustCompile(""))
	if want, got := "", stringFromList(l); want != got {
		t.Errorf("clear: got %#v; want %#v", got, want)
	}
}

func TestDel(t *testing.T) {
	// init
	l := list.New()
	for _, v := range []int{1, 2, 3, 4, 5} {
		l.PushBack(v)
	}

	// del same index multiple times
	del(l, []int{3, 3})
	if want, got := "1, 2, 4, 5", stringFromList(l); want != got {
		t.Errorf("del: got %#v; want %#v", got, want)
	}

	// del consecutive & non-consecutive indices
	del(l, []int{1, 2, 4})
	if want, got := "4", stringFromList(l); want != got {
		t.Errorf("del: got %#v; want %#v", got, want)
	}
}

func TestGetIndex(t *testing.T) {
	l := list.New()
	for _, v := range []int{1, 2, 3, 4, 5} {
		l.PushBack(v)
		if got := getIndex(l, v).Value.(int); got != v {
			t.Errorf("getIndex: got %#v; want %#v", got, v)
		}
	}
}

func TestLs(t *testing.T) {
	// ls empty list
	b := &bytes.Buffer{}
	l := list.New()
	ls(l, b)
	if want, got := "", b.String(); want != got {
		t.Errorf("ls: got %#v; want %#v", got, want)
	}

	// ls non-empty list
	b.Reset()
	l.PushBack("hello")
	l.PushBack("world")
	ls(l, b)
	if want, got := "hello\nworld\n", b.String(); want != got {
		t.Errorf("ls: got %#v; want %#v", got, want)
	}
}

func TestParseInts(t *testing.T) {
	// parse empty list
	got, _ := parseInts([]string{})
	if want := []int{}; fmt.Sprintf("%#v", want) != fmt.Sprintf("%#v", got) {
		t.Errorf("parseInts: got %#v; want %#v", got, want)
	}

	// parse non-empty list
	got, _ = parseInts([]string{"-1", "0", "1"})
	if want := []int{-1, 0, 1}; fmt.Sprintf("%#v", want) !=
		fmt.Sprintf("%#v", got) {
		t.Errorf("parseInts: got %#v; want %#v", got, want)
	}

	// parse invalid strings
	_, err := parseInts([]string{"one"})
	if want, got := "one", err.Error(); want != got {
		t.Errorf("parseInts: got %#v; want %#v", got, want)
	}
}

// testStartServer is a helper function for testing startServer.
func testStartServer(t *testing.T, addr string, status int, out, err string,
	outbuf, errbuf *bytes.Buffer) {
	c := startServer(addr, false)
	ioCmp(t, "startServer", <-c, status, out, err, outbuf, errbuf)
}

func TestStartServer(t *testing.T) {
	// redirect stdout and stderr to buffers
	outbuf, errbuf := &bytes.Buffer{}, &bytes.Buffer{}
	stdout, stderr = outbuf, errbuf

	// test invalid addr
	testStartServer(t, "bogus", 1,
		"", "listen tcp: missing port in address bogus\n", outbuf, errbuf)

	// test available addr
	go testStartServer(t, testAddr, 0, "", "", outbuf, errbuf)

	// test unavailable addr
	go testStartServer(t, testAddr, 1,
		"", "listen tcp :1234: bind: address already in use\n",
		outbuf, errbuf)

	sendCommand(testAddr, "kill")

	// reset stdout and stderr to defaults
	stdout, stderr = os.Stdout, os.Stderr
}

func TestWriteStatus(t *testing.T) {
	buf := &bytes.Buffer{}
	writeStatus(buf, stateStop, nil)
	if want, got := "stopped\n", buf.String(); want != got {
		t.Errorf("writeStatus: got %#v; want %#v", got, want)
	}

	buf.Reset()
	writeStatus(buf, statePause, []string{"echo"})
	if want, got := "paused: echo\n", buf.String(); want != got {
		t.Errorf("writeStatus: got %#v; want %#v", got, want)
	}

	buf.Reset()
	writeStatus(buf, statePlay, []string{"echo"})
	if want, got := "playing: echo\n", buf.String(); want != got {
		t.Errorf("writeStatus: got %#v; want %#v", got, want)
	}
}
