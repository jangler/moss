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
	Usage: moss [<option> ...] [<cmd> [<arg> ...]]

	If invoked with the -start option, a moss server is started.
	Otherwise, the given command and its arguments are sent to the server.
	Specifying no command is equivalent to specifying the 'status' command.

	On server start, commands are read from ~/.mossrc or ~/.config/mossrc.

	Commands:
	  add <item> ...        append an item to the playlist
	  assoc <regexp> <cmd>  associate cmd with items that match regexp
	  del <index> ...       remove items from the playlist
	  insert <item> ...     insert an item after the current item
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

	Options:
	  -addr=":7781": address to connect to
	  -start=false: start server instead of sending command
	  -version=false: display version information and exit
