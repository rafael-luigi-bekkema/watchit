# WatchIt

A program that runs a command when it detects file changes.

Only runs on Linux for now.

## Example
```
watchit -- echo {}
```

## Installation

With Go available:
```
go install github.com/rafael-luigi-bekkema/watchit
```

Or to install to `/usr/local/bin/watchit`:
```
make && sudo make install
```

## Usage
```
WatchIt is a program that can run a command in response to file changes. Also known as a 'file watcher'.

Usage:
  watchit CMD [flags]

Examples:
watchit -g '*.txt' -- stat {}
watchit --mode service  -- go run .
watchit --mode generate -g '*.go' -- go generate
watchit -s $SHELL -- 'echo {} && date'


Flags:
  -e, --events strings       Filesystem events to watch, comma separated. Options are: create,write,remove,rename,chmod.
  -g, --glob stringArray     Filename patterns to filter to (supports ** and {opt1,opt2}).
  -h, --help                 help for watchit
  -i, --ignore stringArray   Filename patterns to ignore (see --glob).
  -m, --mode string          There are 3 modes: react, generate, and service. react runs the command for each changed file, generate runs the command on startup and then for each changed file, service runs the command on startup, and restarts the command for each burst (within 1ms) of changed files. (default "react")
  -p, --path string          Set working directory.
      --placeholder string   String to use as placeholder. If the placeholder appears in the command, it will be replaced with the filename associated with the triggering event. (default "{}")
  -s, --shell string         Shell to use for command. Default is to run the command directly.
  -v, --verbose              Output more information.
```
