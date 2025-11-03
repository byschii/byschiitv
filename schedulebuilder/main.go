package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// load .env (optional). If .env is missing we continue; env vars from the
	// environment are still used (e.g. HOST_MEDIA_PATH).
	if err := godotenv.Load("../.env"); err != nil {
		// Not fatal: just continue. Uncomment next line to debug missing file.
		// fmt.Fprintln(os.Stderr, "No .env file loaded:", err)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
