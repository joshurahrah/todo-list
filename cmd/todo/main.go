package main

import (
	"fmt"
	"os"

	"todo/internal/todo"
	"todo/internal/tui"
)

func main() {
	store := todo.NewMemStore()
	svc := todo.NewService(store)

	if err := tui.Run(svc); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
