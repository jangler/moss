package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

var version = []int{0, 0, 0}

func versionFunc() {
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(1)
	}
	fmt.Printf("%s version %d.%d.%d %s/%s\n", os.Args[0], version[0],
		version[1], version[2], runtime.GOOS, runtime.GOARCH)
}

func init() {
	commands["version"] = &Command{
		Usage: "version",
		Help: `
Print version information.`,
		Func: versionFunc,
	}
}
