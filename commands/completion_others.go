//go:build !darwin && !linux && !freebsd && !openbsd
// +build !darwin,!linux,!freebsd,!openbsd

package commands

import (
	"github.com/posener/complete"
	"github.com/symfony-cli/console"
)

func autocompleteComposerWrapper(context *console.Context, args complete.Args) []string {
	return []string{}
}

func autocompleteSymfonyConsoleWrapper(context *console.Context, args complete.Args) []string {
	return []string{}
}
