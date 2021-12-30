//go:build !windows
// +build !windows

package inotify

import "github.com/syncthing/notify"

func Watch(path string, c chan<- notify.EventInfo, events ...notify.Event) error {
	return simpleWatch(path, c, events...)
}
