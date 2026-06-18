package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type EventHandler func(string) error

type Watcher struct {
	watcher *fsnotify.Watcher
	events  chan string
	done    chan struct{}
}

func New() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher: w,
		events:  make(chan string),
		done:    make(chan struct{}),
	}, nil
}

func (w *Watcher) Add(path string) error {
	return w.watcher.Add(path)
}

func (w *Watcher) AddRecursive(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New(root + " is not a directory")
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Watch(debounceDelay time.Duration, handler EventHandler) {
	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if !shouldIgnore(event.Name) {
						w.events <- event.Name
					}
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				os.Stderr.WriteString("watcher error: " + err.Error() + "\n")
			}
		}
	}()

	timer := time.NewTimer(debounceDelay)
	timer.Stop()
	for {
		select {
		case <-w.events:
			timer.Reset(debounceDelay)
		case <-timer.C:
			if err := handler("change"); err != nil {
				os.Stderr.WriteString("rebuild error: " + err.Error() + "\n")
			}
			timer.Stop()
		case <-w.done:
			return
		}
	}
}

func shouldIgnore(name string) bool {
	base := filepath.Base(name)
	if strings.HasPrefix(base, ".fz_objs") || strings.HasPrefix(base, ".fz_cache") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".o" || ext == ".out" || ext == ".exe" {
		return true
	}
	return false
}

func (w *Watcher) Close() {
	close(w.done)
	w.watcher.Close()
}