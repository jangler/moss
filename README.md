Moss
====
Something like [MPD](http://www.musicpd.org/), but allowing arbitrary commands
to be executed instead of relying on specific playback libraries and plugins.

This project is alpha status and may experience breaking changes.

Installation
------------
	go get -u github.com/jangler/moss

Usage
-----
	Usage: moss [<option> ...] <cmd> [<arg> ...]

	If invoked with the -start option, a moss server is started in the foreground.
	Otherwise, the given command and its arguments are sent to the server.

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

	Options:
	  -addr=":7781": address to connect to
	  -start=false: start server instead of sending command
	  -version=false: display version information and exit
