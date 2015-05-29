package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

const version = "0.0.0"

var (
	addrFlag    string
	startFlag   bool
	versionFlag bool
)

// die prints an error message to standard error and exists with nonzero
// status.
func die(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// usage prints usage information about the program to standard error.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [<option> ...] <cmd> [<arg> ...]\n",
		os.Args[0])

	fmt.Fprintln(os.Stderr, `
If invoked with the -start option, a moss server is started in the foreground.
Otherwise, the given command and its arguments are sent to the server.`)

	fmt.Fprint(os.Stderr, `
Commands:
  add <cmd> [<arg> ...]  add a command to the current list
  del <index> ...        remove commands from the current list
  kill                   stop the server
  ls                     print the current list
  next                   step forward in the list
  pause                  suspend the current command
  play                   start or resume the current command
  prev                   step backward in the list
  status                 print the current status and command
  stop                   kill the current command
  toggle                 toggle between play and pause states
`)

	fmt.Fprintln(os.Stderr, "\nOptions:")
	flag.PrintDefaults()
}

func main() {
	// init flags
	flag.Usage = usage
	flag.StringVar(&addrFlag, "addr", ":7781", "address to connect to")
	flag.BoolVar(&startFlag, "start", false, "start server instead of "+
		"sending command")
	flag.BoolVar(&versionFlag, "version", false, "display version "+
		"information and exit")
	flag.Parse()

	// do what feels right
	if versionFlag {
		fmt.Printf("%s version %s %s/%s\n", os.Args[0], version, runtime.GOOS,
			runtime.GOARCH)
	} else if startFlag {
		startServer()
	} else if flag.NArg() == 0 {
		// confused user
		flag.Usage()
		os.Exit(1)
	} else {
		sendCommand(flag.Args()...)
	}
}
