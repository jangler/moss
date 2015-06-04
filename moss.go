package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const version = "0.1.0"

var (
	addrFlag    string
	startFlag   bool
	stdinFlag   bool
	versionFlag bool
)

// usage prints usage information about the program to standard error.
func usage() {
	fmt.Fprintf(stderr, "Usage: %s [<option> ...] [<cmd> [<arg> ...]]\n",
		os.Args[0])

	fmt.Fprintln(stderr, `
If invoked with the -start option, a moss server is started.
Otherwise, the given command and its arguments are sent to the server.
Specifying no command is equivalent to specifying the 'status' command.

On server start, commands are read from ~/.mossrc or ~/.config/mossrc.`)

	fmt.Fprint(stderr, `
Commands:
  add <item> ...        append items to the playlist
  assoc <regexp> <cmd>  associate cmd with items that match regexp
  clear [<regexp>]      clear playlist, or remove items matching regexp
  del <index> ...       remove items from the playlist
  insert <item> ...     insert items after the current item
  kill                  stop the server and current command
  ls                    print the current playlist
  lsassoc               print the list of command associations
  mv <index> <index>    move an item from one index to another
  next                  step forward in the playlist
  pause                 suspend the current command
  play [<index>]        resume current command or start command at index
  prev                  step backward in the playlist
  status                print the current status and command
  stop                  kill the current command
  toggle                toggle between play and pause states
  unassoc <regexp>      remove the command association for regexp
`)

	fmt.Fprintln(stderr, "\nOptions:")
	flag.PrintDefaults()
}

// readLines reads each line from r and returns the lines as a slice of
// strings.
func readLines(r io.Reader) []string {
	a := make([]string, 0)
	b := bufio.NewReader(r)
	for {
		line, err := b.ReadString('\n')
		if len(line) > 0 {
			a = append(a, strings.TrimSuffix(line, "\n"))
		}
		if err != nil {
			break
		}
	}
	return a
}

func main() {
	// init flags
	flag.Usage = usage
	flag.StringVar(&addrFlag, "addr", ":7781", "address to connect to")
	flag.BoolVar(&startFlag, "start", false, "start server instead of "+
		"sending command")
	flag.BoolVar(&stdinFlag, "stdin", false, "read additional arguments from "+
		"standard input")
	flag.BoolVar(&versionFlag, "version", false, "display version "+
		"information and exit")
	flag.Parse()

	// get args, reading from stdin if specified
	args := flag.Args()
	if stdinFlag {
		args = append(args, readLines(os.Stdin)...)
	}

	// do what feels right
	if versionFlag {
		fmt.Fprintf(stdout, "%s version %s %s/%s\n", os.Args[0], version,
			runtime.GOOS, runtime.GOARCH)
	} else if startFlag {
		c := startServer(addrFlag, true)
		os.Exit(<-c)
	} else if len(args) == 0 {
		os.Exit(sendCommand(addrFlag, "status"))
	} else {
		os.Exit(sendCommand(addrFlag, args...))
	}
}
