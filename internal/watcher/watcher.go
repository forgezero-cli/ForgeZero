/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
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
			if shouldIgnore(path) {
				return filepath.SkipDir
			}
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
	if !timer.Stop() {
		<-timer.C
	}

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(debounceDelay)
	}

	for {
		select {
		case <-w.events:
			resetTimer()
		case <-timer.C:
			if err := handler("change"); err != nil {
				os.Stderr.WriteString("rebuild error: " + err.Error() + "\n")
			}
		case <-w.done:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func shouldIgnore(name string) bool {
	parts := strings.Split(filepath.ToSlash(name), "/")
	for _, p := range parts {
		if strings.HasPrefix(p, ".fz_objs") || strings.HasPrefix(p, ".fz_cache") {
			return true
		}
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
