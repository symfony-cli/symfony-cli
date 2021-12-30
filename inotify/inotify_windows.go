package inotify

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/syncthing/notify"
)

func Watch(path string, c chan<- notify.EventInfo, events ...notify.Event) error {
	if filepath.Base(path) == "..." {
		// This means a recursive watch
		return simpleWatch(path, c, events...)
	} else if fi, err := os.Stat(path); err != nil {
		return errors.WithStack(err)
	} else if fi.IsDir() {
		return simpleWatch(path, c, events...)
	}

	wrappedCh := make(chan notify.EventInfo, 10)
	if err := notify.Watch(filepath.Dir(path), wrappedCh, events...); err != nil {
		return errors.WithStack(err)
	}

	go func() {
		for {
			e := <-wrappedCh
			if e == nil {
				continue
			}
			if path != e.Path() {
				continue
			}
			c <- e
		}
	}()

	return nil
}
