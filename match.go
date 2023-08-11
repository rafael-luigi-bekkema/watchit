package main

import (
	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
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
		if ok, _ := doublestar.PathMatch(glob, filename); ok {
			return false
		}
	}
	if len(w.globs) == 0 {
		return true // No globs so match everything (that isn't ignored)
	}
	for _, glob := range w.globs {
		if ok, _ := doublestar.PathMatch(glob, filename); ok {
			return true
		}
	}
	return false
}
