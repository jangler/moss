package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

func killFunc() {
	// exit on bad usage
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		die("kill: server not running")
	}
	defer conn.Close()
	fmt.Fprint(conn, "kill")
}

func init() {
	commands["kill"] = &Command{
		Usage: "kill",
		Help: `
Kill the mq server.`,
		Func: killFunc,
	}
}
