package inotify

import "github.com/syncthing/notify"

// Create, Remove, Write and Rename are the only event values guaranteed to be
// present on all platforms.
const (
	Create = notify.Create
	Remove = notify.Remove
	Write  = notify.Write
	Rename = notify.Rename

	// All is handful alias for all platform-independent event values.
	All = Create | Remove | Write | Rename
)

type EventInfo = notify.EventInfo

func Stop(c chan<- EventInfo) {
	notify.Stop(c)
}

func simpleWatch(path string, c chan<- EventInfo, events ...notify.Event) error {
	return notify.Watch(path, c, events...)
}
