package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type screenState int

const (
	screenDirInput screenState = iota
	screenMain
)

type model struct {
	state      screenState
	dirInput   DirInputScreen
	mainScreen MainScreen

	width  int
	height int
}

func initialModel() model {
	return model{
		state:    screenDirInput,
		dirInput: newDirInputScreen(),
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// pass dimensions to current screen
		if m.state == screenDirInput {
			m.dirInput.width = m.width
			m.dirInput.height = m.height
		}
		if m.state == screenMain {
			m.mainScreen.width = m.width
			m.mainScreen.height = m.height
		}
	}

	// global quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "ctrl+c" || (m.state == screenMain && msg.String() == "q") {
			return m, tea.Quit
		}
	}

	switch m.state {
	case screenDirInput:
		return m.updateDirInput(msg)
	case screenMain:
		return m.updateMain(msg)
	}
	return m, nil
}

func (m model) updateDirInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.dirInput.update(msg)

	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
		if valid, path := m.dirInput.validate(); valid {
			m.state = screenMain
			m.mainScreen = newMainScreen(path)
		}
	}
	return m, cmd
}

func (m model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	// if user pressed 'e', go back to directory input screen so they can edit
	// the base directory. Prefill the input with the current base dir so
	// confirming will create a new main screen (which re-runs the scan).
	if key, ok := msg.(tea.KeyMsg); ok {
		// don't allow 'e' to trigger directory edit while the search box is active
		if key.String() == "e" && !m.mainScreen.search.active {
			d := newDirInputScreen()
			// prefill with current base dir
			d.input.SetValue(m.mainScreen.baseDir)
			d.input.Placeholder = m.mainScreen.baseDir
			d.baseDir = m.mainScreen.baseDir
			m.dirInput = d
			m.state = screenDirInput
			return m, nil
		}
	}

	cmd := m.mainScreen.update(msg)
	return m, cmd
}

func (m model) View() string {

	/*switch m.state {
	case screenDirInput:
		return m.dirInput.view()
	case screenMain:
		return m.mainScreen.view()
	}*/
	return drawBorder(m.width, m.height)
}

func drawBorder(width, height int) string {
	titleHeight := 1
	title := " Schedule Builder (ctrl+c to quit) "

	s := "\n\n"
	// top border
	s += "┌" + repeat("─", width-2) + "┐\n"
	// title
	s += "│" + title + strings.Repeat(" ", width-2-len(title)) + "│\n"
	s += "├" + repeat("─", width-2) + "┤\n"

	// empty space
	for i := 0; i < height-titleHeight-1-3; i++ {
		s += "│" + repeat(" ", width-2) + "│\n"
	}

	// bottom border
	s += "└" + repeat("─", width-2) + "┘\n"
	return s
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
