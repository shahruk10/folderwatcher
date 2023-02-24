// Copyright (2023 -- present) Shahruk Hossain <shahruk10@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//		 http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ==============================================================================

package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type Config struct {
	IncludeFolders []string `yaml:"include_folders"`
	ExcludeFolders []string `yaml:"exclude_folders"`
}

func (cfg *Config) Validate() error {
	if len(cfg.IncludeFolders) == 0 {
		return fmt.Errorf("no folders to watch specified")
	}

	return nil
}

// Op describes a set of file operations.
type Op uint32

const (
	CreateOp Op = 1 << iota
	WriteOp
	RemoveOp
	RenameOp
	ChmodOp
)

// An Event is triggered when one or more file operations have been detected in
// the folders being watched.
type Event struct {
	*fsnotify.Event
	time time.Time
}

func (e *Event) HasOp(op Op) bool {
	return e.Has(fsnotify.Op(op))
}

func (e *Event) IsSameWriteEventAs(e0 *Event) bool {
	opPairs := [][]fsnotify.Op{
		{fsnotify.Create, fsnotify.Create},
		{fsnotify.Create, fsnotify.Write},
		{fsnotify.Write, fsnotify.Create},
		{fsnotify.Write, fsnotify.Write},
	}

	consecutiveWriteEvent := false
	for _, p := range opPairs {
		consecutiveWriteEvent = consecutiveWriteEvent || (e0.Has(p[0]) && e.Has(p[1]))
	}

	elapsedTime := e.time.Sub(e0.time)

	return elapsedTime < time.Second && consecutiveWriteEvent
}

type Callback = func(ctx context.Context, logger *logrus.Logger, e Event) error

type Watcher interface {
	AddFolders(folderPaths ...string) error
	AddCallbacks(callbacks ...Callback) error
	Watch(ctx context.Context) error
	Close() error
}

type FSNotifyWatcher struct {
	*fsnotify.Watcher

	logger    *logrus.Logger
	cfg       Config
	callbacks []Callback
}

func (w *FSNotifyWatcher) AddFolders(folderPaths ...string) error {
	for _, folder := range folderPaths {
		if err := w.Add(folder); err != nil {
			return fmt.Errorf("%q: %w", folder, err)
		}
	}

	return nil
}

func (w *FSNotifyWatcher) AddCallbacks(callbacks ...Callback) error {
	for _, cb := range callbacks {
		if cb == nil {
			return fmt.Errorf("nil callback function")
		}

		w.callbacks = append(w.callbacks, cb)
	}

	return nil
}

func (w *FSNotifyWatcher) Watch(ctx context.Context) error {
	eventLog := make(map[string]*Event)

	for {
		t0 := time.Now()
		purge := make([]string, 0, len(eventLog))
		for name, e := range eventLog {
			if t0.Sub(e.time) > 30*time.Second {
				purge = append(purge, name)
			}
		}

		for _, name := range purge {
			delete(eventLog, name)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case e, ok := <-w.Events:
			if !ok {
				return nil
			}

			newEvent := Event{Event: &e, time: time.Now()}
			ignore := false

			w.logger.Debugf("received event: %s", e)

			prevEvent, ok := eventLog[e.Name]
			if ok {
				ignore = newEvent.IsSameWriteEventAs(prevEvent)
			}

			eventLog[e.Name] = &newEvent

			if ignore {
				w.logger.Infof("ignoring consecutive write events for %q", newEvent.Name)
				continue
			}

			for i, callback := range w.callbacks {
				if err := callback(ctx, w.logger, newEvent); err != nil {
					w.logger.Errorf("applying callback[%d]: %v", i, err)
				}
			}

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}

			if err != nil {
				w.logger.Error("encountered error: %v", err)
			}
		}
	}
}

func (w *FSNotifyWatcher) Close() error {
	w.logger.Info("Closing watcher")
	return w.Watcher.Close()
}

func New(logger *logrus.Logger, cfg Config) (Watcher, error) {
	wInternal, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	w := &FSNotifyWatcher{Watcher: wInternal, logger: logger, cfg: cfg}

	return w, nil
}
