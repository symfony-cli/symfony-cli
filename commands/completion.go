//go:build darwin || linux || freebsd || openbsd

package commands

import (
	"embed"

	"github.com/symfony-cli/console"
)

// completionTemplates holds our custom shell completions templates.
//
//go:embed resources/completion.*
var completionTemplates embed.FS

func init() {
	// override console completion templates with our custom ones
	console.CompletionTemplates = completionTemplates
}
