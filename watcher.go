package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
)

var defaultNoWatches = []string{"**/.git", "**/vendor", "**/node_modules"}

type mode int

const (
	modeGenerate mode = iota
	modeReact
	modeService
)

type watcher struct {
	command     []string
	globs       []string
	ignoreGlobs []string
	noWatch     []string
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

func newWatcher() (*watcher, error) {
	var err error
	var w watcher
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w.runcmd = make(chan string)
	w.actions = make(chan action)
	return &w, nil
}

func (w *watcher) addRecursive(workDir, path string) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		relpath, _ := filepath.Rel(workDir, path)
		for _, glob := range w.noWatch {
			if ok, _ := doublestar.PathMatch(glob, relpath); ok {
				return filepath.SkipDir
			}
		}

		if w.verbose {
			slog.Info("adding watch for", "path", path)
		}
		if err := w.watcher.Add(path); err != nil {
			return err
		}
		return nil
	})
}

type actionType int

const (
	actionAdd actionType = iota
)

type action struct {
	typ     actionType
	payload string
}

func (w *watcher) eventLoop(workDir string) error {
	for action := range w.actions {
		switch action.typ {
		case actionAdd:
			if err := w.addRecursive(workDir, action.payload); err != nil {
				return err
			}
		}
	}
	return nil
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

	if err := w.addRecursive(workDir, path); err != nil {
		return err
	}

	go w.listen(workDir)
	go w.cmdLoop()
	if w.mode == modeService || w.mode == modeGenerate {
		w.runcmd <- ""
	}
	return w.eventLoop(workDir)
}
