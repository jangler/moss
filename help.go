package main

import (
	"flag"
	"fmt"
	"os"
)

func helpFunc() {
	if flag.NArg() == 1 {
		flag.Usage()
	} else if flag.NArg() == 2 {
		if cmd, ok := commands[flag.Arg(1)]; ok {
			fmt.Printf("Usage: mq %s\n", cmd.Usage)
			fmt.Println(cmd.Help)
		} else {
			die("help: no such command: " + flag.Arg(1))
		}
	} else {
		flag.Usage()
		os.Exit(1)
	}
}

func init() {
	commands["help"] = &Command{
		Usage: "help [<command>]",
		Help: `
If 'command' is given, print usage and help information for that command.
Otherwise, print general usage information.`,
		Func: helpFunc,
	}
}
