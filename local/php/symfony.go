package php

import "os"

// ComposerExecutor returns an Executor prepared to run Symfony Console
func SymonyConsoleExecutor(args []string) *Executor {
	consolePath := "bin/console"
	if _, err := os.Stat("app/console"); err == nil {
		consolePath = "app/console"
	}

	return &Executor{
		BinName: "php",
		Args:    append([]string{"php", consolePath}, args...),
	}
}
