# WatchIt

A program that runs a command when it detects file changes.

Only runs on Linux for now.

## Why?

- Restart dev server when source files change
- Re-generate asset based on changes in other assets
- Run a linter / formatter when source file changes

As a fully static binary `watchit` can easily be deployed anywhere.

## Example
```
watchit react -- echo {}
```

## Installation

With Go available:
```
go install github.com/rafael-luigi-bekkema/watchit@latest
```

Or to **go install** from a clone:
```
make install
```

## Usage
```
Run the command when a file changes.

Usage:
  watchit react CMD [flags]

Aliases:
  react, r

Flags:
  -e, --events strings         Filesystem events to watch, comma separated. Options are: create,write,remove,rename,chmod.
  -g, --glob stringArray       Filename patterns to filter to (supports ** and {opt1,opt2}).
  -h, --help                   help for react
  -i, --ignore stringArray     Filename patterns to ignore (see --glob).
  -n, --no-watch stringArray   Patterns for directories that should not be watched. --glob and --ignore do not affect watched directories.
  -p, --path string            Set working directory.
      --placeholder string     String to use as placeholder. If the placeholder appears in the command, it will be replaced with the filename associated with the triggering event. (default "{}")
  -s, --shell string           Shell to use for command. Default is to run the command directly.
  -v, --verbose                Output more information.

Usage:
  watchit generate CMD [flags]

Usage:
  watchit service CMD [flags]
```

## Environment variables

**WATCHIT_NO_WATCH** can be used to set patterns for directories that should not be watched.  
For example add to your .bashrc / .zshrc:
```
export WATCHIT_NO_WATCH=**/.git:**/node_modules:**/vendor
```

You can use something like **direnv** to manage this on a per project basis.


## Glob

For glob pattern rules see:
<https://github.com/bmatcuk/doublestar#patterns>


## Modes

There are 3 modes: react, generate and service. The mode is the first argument.

**react**  

Runs the command when a file changes.

**generate**  

Runs the command upon start and when a file changes.

**service**  

Runs the command upon start and restarts the command after each burst (within 1 ms) of file changes.

## Shell

Normally the command runs directly without a shell. So the first argument is the main command, and further arguments as passed as arguments to the command.  
To use things like && or > you can pass a shell, which will then be used to run the command. For example:
```
watchit react -s $SHELL -- 'echo {} changed && date'
# or
watchit react -s bash -- 'echo {} changed && date'
```
