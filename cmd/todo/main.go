package main

import (
	"fmt"
	"os"

	"todo/internal/todo"
	"todo/internal/tui"
)

func main() {
	path, err := todo.DefaultStatePath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving state path:", err)
		os.Exit(1)
	}

	ws, err := todo.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading state:", err)
		os.Exit(1)
	}

	if err := tui.Run(ws, path); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
