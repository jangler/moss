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

// usage prints usage information about the program to standard error.
func usage() {
	fmt.Fprintf(stderr, "Usage: %s [<option> ...] [<cmd> [<arg> ...]]\n",
		os.Args[0])

	fmt.Fprintln(stderr, `
If invoked with the -start option, a moss server is started.
Otherwise, the given command and its arguments are sent to the server.
Specifying no command is equivalent to specifying the 'status' command.`)

	fmt.Fprint(stderr, `
Commands:
  add <cmd> [<arg> ...]     append a command to the playlist
  del <index> ...           remove commands from the playlist
  insert <cmd> [<arg> ...]  insert a command after the current command
  kill                      stop the server and current command
  ls                        print the current playlist
  next                      step forward in the playlist
  pause                     suspend the current command
  play [<index>]            resume current command or start command at index
  prev                      step backward in the playlist
  status                    print the current status and command
  stop                      kill the current command
  toggle                    toggle between play and pause states
`)

	fmt.Fprintln(stderr, "\nOptions:")
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
		fmt.Fprintf(stdout, "%s version %s %s/%s\n", os.Args[0], version,
			runtime.GOOS, runtime.GOARCH)
	} else if startFlag {
		c := startServer(addrFlag)
		os.Exit(<-c)
	} else if flag.NArg() == 0 {
		os.Exit(sendCommand(addrFlag, "status"))
	} else {
		os.Exit(sendCommand(addrFlag, flag.Args()...))
	}
}
