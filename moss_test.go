package main

import (
	"bytes"
	"container/list"
	"fmt"
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
	l.PushBack([]string{"echo", "hello"})
	l.PushBack([]string{"echo", "world"})
	ls(l, b)
	if want, got := "echo hello\necho world\n", b.String(); want != got {
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
	if want := []int{-1, 0, 1}; fmt.Sprintf("%#v", want) != fmt.Sprintf("%#v", got) {
		t.Errorf("parseInts: got %#v; want %#v", got, want)
	}

	// parse invalid strings
	_, err := parseInts([]string{"one"})
	if want, got := "one", err.Error(); want != got {
		t.Errorf("parseInts: got %#v; want %#v", got, want)
	}
}
