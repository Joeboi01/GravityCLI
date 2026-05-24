package main

import (
	"GravityCLI/cmd"
	"GravityCLI/internal/console"
)

func main() {
	console.EnableVirtualTerminalProcessing()
	cmd.Execute()
}

