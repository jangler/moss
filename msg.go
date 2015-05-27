package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func sendMsg(cmd, msg string) {
	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		die(cmd + ": server not running")
	}
	defer conn.Close()
	fmt.Fprint(conn, msg)
}

func init() {
	commands["kill"] = &Command{
		Usage: "kill",
		Help: `
Kill the mq server.`,
		Func: func() {
			if flag.NArg() > 1 {
				flag.Usage()
				os.Exit(1)
			}
			sendMsg("kill", "kill")
		},
	}

	commands["play"] = &Command{
		Usage: "play [<index>]",
		Help: `
Run the command at position 'index' in the queue, or run the first command if
no index is specified.`,
		Func: func() {
			if flag.NArg() > 2 {
				flag.Usage()
				os.Exit(1)
			}
			sendMsg("push", strings.Join(flag.Args(), " "))
		},
	}

	commands["push"] = &Command{
		Usage: "push <command> [<arg> ...]",
		Help: `
Push a command onto the queue.`,
		Func: func() {
			if flag.NArg() < 2 {
				flag.Usage()
				os.Exit(1)
			}
			sendMsg("push", strings.Join(flag.Args(), " "))
		},
	}

	commands["stop"] = &Command{
		Usage: "stop",
		Help: `
Kill the currently running command.`,
		Func: func() {
			if flag.NArg() != 1 {
				flag.Usage()
				os.Exit(1)
			}
			sendMsg("stop", "stop")
		},
	}
}
