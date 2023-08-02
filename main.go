package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

type watcher struct {
	command     []string
	globs       []string
	ignoreGlobs []string
	events      []fsnotify.Op
	workingDir  string
	shell       string
	placeholder string

	verbose bool

	watcher *fsnotify.Watcher

	runcmd  chan string
	actions chan action
	mode    mode
}

type mode int

const (
	modeGenerate mode = iota
	modeReact
	modeService
)

// match reports whether filename matches according to the passed in filters
func (w *watcher) match(op fsnotify.Op, filename string) bool {
	// check if operation is in events list
	var hasOp bool
	for _, evt := range w.events {
		if op.Has(evt) {
			hasOp = true
			break
		}
	}
	if w.events != nil && !hasOp {
		return false
	}
	for _, glob := range w.ignoreGlobs {
		if ok, _ := doublestar.Match(glob, filename); ok {
			return false
		}
	}
	if len(w.globs) == 0 {
		return true // No globs so match everything (that isn't ignored)
	}
	for _, glob := range w.globs {
		if ok, _ := doublestar.Match(glob, filename); ok {
			return true
		}
	}
	return false
}

func (w *watcher) buildCmd(filename string) []string {
	// copy original command and replace placeholder with changed filename
	cmd := make([]string, len(w.command))
	copy(cmd, w.command)
	if w.placeholder != "" && w.mode != modeService {
		for i, val := range cmd {
			if strings.Contains(val, w.placeholder) {
				cmd[i] = strings.ReplaceAll(val, w.placeholder, filename)
			}
		}
	}

	if w.shell != "" {
		cmd = append([]string{w.shell, "-c"}, strings.Join(cmd, " "))
	}
	return cmd
}

func (w *watcher) runCmd(filename string) (*exec.Cmd, error) {
	cmd := w.buildCmd(filename)
	ccmd := exec.Command(cmd[0], cmd[1:]...)
	ccmd.Stderr = os.Stderr
	ccmd.Stdout = os.Stdout
	err := ccmd.Start()
	if err != nil {
		return nil, err
	}
	return ccmd, nil
}

func (w *watcher) addRecursive(path string) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		if w.verbose {
			log.Printf("adding watch for %s", path)
		}
		if err := w.watcher.Add(path); err != nil {
			return err
		}
		return nil
	})
}

// cmdLoop is the command runner with burst protection
func (w *watcher) cmdLoop() {
	var prevCmd *exec.Cmd
outer:
	fnames := map[string]struct{}{
		<-w.runcmd: {},
	}

inner:
	select {
	case newcmd := <-w.runcmd:
		fnames[newcmd] = struct{}{}
		goto inner
	case <-time.After(time.Millisecond):
		for fname := range fnames {
			if prevCmd != nil {
				if err := prevCmd.Process.Signal(syscall.SIGTERM); err != nil {
					log.Printf("could not send SIGTERM: %s", err)
				}
				if err := prevCmd.Wait(); err != nil {
					log.Printf("command failed: %s", err)
				}
			}

			cmd, err := w.runCmd(fname)
			if err != nil {
				log.Printf("failed to run command: %s", err)
				goto outer
			}
			if w.mode == modeService {
				prevCmd = cmd
				break
			} else {
				cmd.Wait()
			}
		}
		goto outer
	}
}

// listen listens to filesystem events and triggers actions/commands
func (w *watcher) listen(workDir string) {
start:
	select {
	case event, ok := <-w.watcher.Events:
		if !ok {
			panic("watch error")
		}
		if w.verbose {
			log.Println("event", event)
		}
		eventName, _ := filepath.Rel(workDir, event.Name)
		if w.match(event.Op, eventName) {
			w.runcmd <- eventName
		}
		if event.Has(fsnotify.Create) {
			stat, err := os.Stat(event.Name)
			if err != nil {
				goto start
			}
			if stat.IsDir() {
				w.actions <- action{actionAdd, event.Name}
			}
		}
	case err, ok := <-w.watcher.Errors:
		if !ok {
			panic("watch error")
		}
		log.Println("error", err)
	}

	goto start
}

func (w *watcher) eventLoop() error {
	for action := range w.actions {
		switch action.typ {
		case actionAdd:
			if err := w.addRecursive(action.payload); err != nil {
				return err
			}
		}
	}
	return nil
}

type actionType int

const (
	actionAdd actionType = iota
)

type action struct {
	typ     actionType
	payload string
}

func (w *watcher) run() error {
	defer w.watcher.Close()

	if w.workingDir != "" {
		if err := os.Chdir(w.workingDir); err != nil {
			return err
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	path := workDir

	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	if err := w.addRecursive(path); err != nil {
		return err
	}

	go w.listen(workDir)
	go w.cmdLoop()
	if w.mode == modeService || w.mode == modeGenerate {
		w.runcmd <- ""
	}
	return w.eventLoop()
}

func opsFromS(items []string) (result []fsnotify.Op, err error) {
	for _, item := range items {
		switch item {
		case "create":
			result = append(result, fsnotify.Create)
		case "write":
			result = append(result, fsnotify.Write)
		case "remove":
			result = append(result, fsnotify.Remove)
		case "rename":
			result = append(result, fsnotify.Rename)
		case "chmod":
			result = append(result, fsnotify.Chmod)
		default:
			return nil, fmt.Errorf("unknown event type: %s", item)
		}
	}
	return result, nil
}

func modeFromS(input string) (mode, error) {
	if len(input) == 0 {
		return 0, fmt.Errorf("mode can not be empty")
	}
	switch input[0] {
	case 'r':
		return modeReact, nil
	case 'g':
		return modeGenerate, nil
	case 's':
		return modeService, nil
	default:
		return 0, fmt.Errorf("unknown mode: %q", input)
	}
}

func main() {
	log.SetFlags(0)

	var w watcher
	var events []string
	var mode string
	mainCommand := cobra.Command{
		Use:   "watchit CMD",
		Short: "Run command on file change.",
		Long:  "WatchIt is a program that can run a command in response to file changes. Also known as a 'file watcher'.",
		Example: `watchit -g '*.txt' -- stat {}
watchit --mode service  -- go run .
watchit --mode generate -g '*.go' -- go generate
watchit -s $SHELL -- 'echo {} && date'
`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			w.watcher, err = fsnotify.NewWatcher()
			if err != nil {
				return err
			}
			w.command = args
			w.runcmd = make(chan string)
			w.actions = make(chan action)
			w.events, err = opsFromS(events)
			if err != nil {
				return err
			}
			w.mode, err = modeFromS(mode)
			if err != nil {
				return err
			}
			return w.run()
		},
	}
	flags := mainCommand.Flags()
	flags.StringArrayVarP(&w.globs, "glob", "g", nil, "Filename patterns to filter to (supports ** and {opt1,opt2}).")
	flags.StringArrayVarP(&w.ignoreGlobs, "ignore", "i", nil, "Filename patterns to ignore (see --glob).")
	flags.StringVarP(&w.workingDir, "path", "p", "", "Set working directory.")
	flags.StringVarP(&w.shell, "shell", "s", "", "Shell to use for command. Default is to run the command directly.")
	flags.BoolVarP(&w.verbose, "verbose", "v", false, "Output more information.")
	flags.StringSliceVarP(&events, "events", "e", nil, "Filesystem events to watch, comma separated. Options are: create,write,remove,rename,chmod.")
	flags.StringVar(&w.placeholder, "placeholder", "{}", "String to use as placeholder. If the placeholder appears in the command, it will be replaced with the filename associated with the triggering event.")
	flags.StringVarP(&mode, "mode", "m", "react", "There are 3 modes: react, generate, and service. react runs the command for each changed file, generate runs the command on startup and then for each changed file, service runs the command on startup, and restarts the command for each burst (within 1ms) of changed files.")

	if err := mainCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
