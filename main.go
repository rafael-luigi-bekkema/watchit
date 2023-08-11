package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

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

func makeCmd(name string, mode mode, short string, w *watcher) *cobra.Command {
	var events []string
	cmd := cobra.Command{
		Use:     name + " CMD",
		Short:   short,
		Args:    cobra.MinimumNArgs(1),
		GroupID: "watchers",
		Aliases: []string{name[0:1]},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			w.command = args
			w.events, err = opsFromS(events)
			if err != nil {
				return err
			}
			w.mode = mode
			return w.run()
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVarP(&w.globs, "glob", "g", nil,
		"Filename patterns to filter to (supports ** and {opt1,opt2}).")
	flags.StringArrayVarP(&w.ignoreGlobs, "ignore", "i", nil,
		"Filename patterns to ignore (see --glob).")
	flags.StringVarP(&w.workingDir, "path", "p", "", "Set working directory.")
	flags.StringVarP(&w.shell, "shell", "s", "",
		"Shell to use for command. Default is to run the command directly.")
	flags.BoolVarP(&w.verbose, "verbose", "v", false, "Output more information.")
	flags.StringSliceVarP(&events, "events", "e", nil,
		"Filesystem events to watch, comma separated. Options are: create,write,remove,rename,chmod.")
	if mode != modeService {
		flags.StringVar(&w.placeholder, "placeholder", "{}",
			"String to use as placeholder. If the placeholder appears in the command, "+
				"it will be replaced with the filename associated with the triggering event.")
	}

	return &cmd
}

func main() {
	log.SetFlags(0)

	w, err := newWatcher()
	if err != nil {
		log.Fatalf("failed to start watcher: %s", err)
	}
	mainCommand := cobra.Command{
		Use:   "watchit",
		Short: "Run command on file change.",
		Long: "WatchIt is a program that can run a command in response to file changes. " +
			"Also known as a 'file watcher'.",
		Example: `watchit react -g '*.txt' -- stat {}
watchit service  -- go run .
watchit generate -g '*.go' -- go generate
watchit react -s $SHELL -- 'echo {} && date'
`,
		SilenceUsage: true,
	}

	mainCommand.AddGroup(&cobra.Group{
		ID:    "watchers",
		Title: "Watchers",
	}, &cobra.Group{
		ID:    "other",
		Title: "Other",
	})

	mainCommand.AddCommand(
		makeCmd("react", modeReact, "Run the command when a file changes.", w),
		makeCmd("generate", modeGenerate, "Run the command upon start and when a file changes.", w),
		makeCmd("service", modeService, "Runs the command upon start and restarts the command "+
			"after each burst (within 1 ms) of file changes.", w),
	)

	if err := mainCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
