Moss
====
Moss is a client/server software that can be used to dynamically create and
execute "playlists" of arbitrary commands. This is mostly useful for audio and
video media, but could be used for any purpose.

A playlist consists of "items", which are simply strings. By default, an item
is executed by passing it as an argument to `/bin/sh -c`, but Moss can easily
be configured to invoke a different program if the item matches a given regular
expression.

The concept for Moss was inspired by
[Music Player Daemon (MPD)](http://www.musicpd.org/), and the interface is
based on that of [mpc](http://www.musicpd.org/clients/mpc/).

Installation
------------
Install via the [go command](http://golang.org/cmd/go/):

	go get -u github.com/jangler/moss

If you use Arch Linux or a derivative, you may also install via the
[AUR package](https://aur.archlinux.org/packages/moss/).

Usage
-----
	Usage: moss [<option> ...] [<cmd> [<arg> ...]]

	If invoked with the -start option, a moss server is started.
	Otherwise, the given command and its arguments are sent to the server.
	Specifying no command is equivalent to specifying the 'status' command.

	On server start, commands are read from ~/.mossrc or ~/.config/mossrc.

	Commands:
	  add <item> ...          append items to the playlist
	  assoc <regexp> <cmd>    associate cmd with items that match regexp
	  clear [<regexp>]        remove items matching regexp, or all [deprecated]
	  del <index> ...         remove items from the playlist
	  index <regexp> ...      print indices of items matching each regexp
	  insert <item> ...       insert items after the current item
	  kill                    stop the server and current command
	  ls                      print the current playlist
	  lsassoc                 print the list of command associations
	  mv <index> ... <index>  move items to the last given index
	  next                    step forward in the playlist
	  pause                   suspend the current command
	  play [<index>]          resume current command or start command at index
	  prev                    step backward in the playlist
	  status [<format>]       print information about the current state
	  stop                    kill the current command
	  toggle                  toggle between play and pause states
	  unassoc <regexp> ...    remove the command association for each regexp

	The status command's <format> argument supports the sequences %c, %f, %i, %n,
	%p, %s, and %t, which are replaced with current command, absolute file path of
	current item, index of current item, playlist length, command process ID,
	status, and current item, respectively.

	Options:
	  -addr=":7781": address to connect to
	  -start=false: start server instead of sending command
	  -stdin=false: read additional arguments from standard input
	  -version=false: display version information and exit

Example
-------
	$ moss -start &
	[1] 20595
	$ moss assoc .\\.mp3$ mpg123
	$ ls
	01_kher_keep.mp3  02_semi-slug.mp3
	$ moss add * && moss ls
	01_kher_keep.mp3
	02_semi-slug.mp3
	$ moss play && moss status
	playing #1/2: mpg123 01_kher_keep.mp3
	$ sleep 10m && moss status
	playing #2/2: mpg123 02_semi-slug.mp3
	$ moss kill
	[1]+  Done                    moss -start
