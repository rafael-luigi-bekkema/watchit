package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

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
