package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// DirInputScreen handles the initial directory input prompt
type DirInputScreen struct {
	input    textinput.Model
	errMsg   string
	baseDir  string
	onAccept func(string) // callback when directory is validated

	// terminal dimensions (populated from WindowSizeMsg)
	width  int
	height int
}

func newDirInputScreen() DirInputScreen {
	ti := textinput.New()
	defaultPath := os.Getenv("HOST_MEDIA_PATH")
	if defaultPath == "" {
		defaultPath = "./byschiitv/media"
	}
	ti.Placeholder = defaultPath
	ti.SetValue(defaultPath)
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 60

	return DirInputScreen{
		input:   ti,
		errMsg:  "",
		baseDir: defaultPath,
	}
}

func (d *DirInputScreen) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

func (d *DirInputScreen) validate() (bool, string) {
	path := d.input.Value()
	info, err := os.Stat(path)
	if err != nil {
		d.errMsg = fmt.Sprintf("path error: %v", err)
		return false, ""
	}
	if !info.IsDir() {
		d.errMsg = "path exists but is not a directory"
		return false, ""
	}
	d.errMsg = ""
	return true, path
}

func (d *DirInputScreen) view() string {
	s := "Enter base directory for videos (press Enter to continue)\n\n"
	s += d.input.View() + "\n\n"
	s += fmt.Sprintf("Detected default (from HOST_MEDIA_PATH or fallback): %s\n", d.input.Placeholder)
	if d.errMsg != "" {
		s += "\nError: " + d.errMsg + "\n"
	}
	s += "\nPress ctrl+c to quit.\n"
	return s
}
