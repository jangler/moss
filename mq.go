package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

type Command struct {
	Usage, Help string
	Func        func()
}

var commands = make(map[string]*Command)

func usage() {
	// print header
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [<arg> ...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nCommands:\n")

	// sort command names
	keys := make([]string, len(commands))
	i := 0
	for key := range commands {
		keys[i] = key
		i++
	}
	sort.StringSlice(keys).Sort()

	// print command list
	for _, key := range keys {
		fmt.Fprintf(os.Stderr, "  %s\n", commands[key].Usage)
	}
}

func main() {
	// init flags
	flag.Usage = usage
	flag.Parse()

	// execute command or die if usage is invalid
	if flag.NArg() == 0 {
		usage()
		os.Exit(1)
	}
	if cmd, ok := commands[flag.Arg(0)]; ok {
		cmd.Func()
	} else {
		usage()
		os.Exit(1)
	}
}
