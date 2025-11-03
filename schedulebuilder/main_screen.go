package main

import (
	"fmt"
	"os"
	"strconv"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// MainScreen handles the dual-column selection interface
type MainScreen struct {
	baseDir       string
	scannedColumn Column
	plannedColumn Column
	activeColumn  int // 0=scanned, 1=planned (currently unused - always 0)
	search        SearchBox
	allScanned    []string // full list before search filter

	// terminal dimensions (populated from WindowSizeMsg)
	width  int
	height int
}

func newMainScreen(baseDir string) MainScreen {
	scanned := scanMedia(baseDir)
	col := newColumn()
	col.setItems(scanned)

	return MainScreen{
		baseDir:       baseDir,
		scannedColumn: col,
		plannedColumn: newColumn(),
		activeColumn:  0,
		search:        newSearchBox(),
		allScanned:    scanned,
	}
}

func (m *MainScreen) update(msg tea.Msg) tea.Cmd {
	// route to search box if active
	if m.search.active {
		return m.handleSearchMode(msg)
	}

	// normal navigation mode
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.search.activate()
			return nil
		case "up", "k":
			m.activeCol().moveCursor(-1)
		case "down", "j":
			m.activeCol().moveCursor(1)
		case "enter", " ":
			m.activeCol().toggleSelection()
		case "left", "h":
			if m.activeColumn > 0 {
				m.activeColumn--
			}
		case "right", "l":
			// disabled: stay on left column
		}
	}
	return nil
}

func (m *MainScreen) handleSearchMode(msg tea.Msg) tea.Cmd {
	cmd := m.search.update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.search.commit()
			query := m.search.value()
			results := sortByJaccard(m.allScanned, query) // sortByLevenshtein(m.allScanned, query)
			m.scannedColumn.setItems(results)
			m.search.deactivate()
			return cmd
		case "esc":
			m.search.deactivate()
			return cmd
		case "up", "k":
			m.search.prevHistory()
			return cmd
		case "down", "j":
			m.search.nextHistory()
			return cmd
		default:
			// live filter as user types
			query := m.search.value()
			if query == "" {
				m.scannedColumn.setItems(m.allScanned)
			} else {
				results := sortByLevenshtein(m.allScanned, query)
				m.scannedColumn.setItems(results)
			}
		}
	}
	return cmd
}

func (m *MainScreen) activeCol() *Column {
	if m.activeColumn == 0 {
		return &m.scannedColumn
	}
	return &m.plannedColumn
}

func (m *MainScreen) view() string {
	s := fmt.Sprintf("Schedule Builder — base dir: %s\n", m.baseDir)

	leftTitle := "Search"
	rightTitle := "Built so far"

	totalWidth := 120
	if c := os.Getenv("COLUMNS"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 40 {
			totalWidth = v
		}
	}
	builtW := 30
	if totalWidth-builtW < 20 {
		builtW = totalWidth - 20
		if builtW < 10 {
			builtW = 10
		}
	}
	lw := totalWidth - builtW
	rw := builtW

	header := fmt.Sprintf("%-*s %-*s\n", lw, leftTitle, rw, rightTitle)
	s += header

	if m.search.active {
		s += m.search.input.View() + "\n"
		s += "(type to narrow results, Enter to apply, Esc to cancel)\n\n"
	}

	rows := m.scannedColumn.len()
	if m.plannedColumn.len() > rows {
		rows = m.plannedColumn.len()
	}

	for i := 0; i < rows; i++ {
		left := ""
		if i < len(m.scannedColumn.items) {
			left = m.scannedColumn.items[i]
		}
		right := ""
		if i < len(m.plannedColumn.items) {
			right = m.plannedColumn.items[i]
		}

		lcur := " "
		if m.activeColumn == 0 && m.scannedColumn.cursor == i {
			lcur = ">"
		}
		rcur := " "
		if m.activeColumn == 1 && m.plannedColumn.cursor == i {
			rcur = ">"
		}

		lchk := " "
		if m.scannedColumn.isSelected(i) {
			lchk = "x"
		}

		leftWidth := lw - 6
		if leftWidth < 8 {
			leftWidth = lw - 4
			if leftWidth < 0 {
				leftWidth = 0
			}
		}
		rightWidth := rw - 4
		if rightWidth < 0 {
			rightWidth = 0
		}

		leftPrinted := truncate(left, leftWidth)
		rightPrinted := truncate(right, rightWidth)

		s += fmt.Sprintf("%s [%s] %-*s %s %-*s\n",
			lcur, lchk, leftWidth, leftPrinted,
			rcur, rightWidth, rightPrinted,
		)
	}

	// show 'e' hint only when not searching
	if m.search.active {
		s += "\n ↑/↓ to move, space/enter to toggle selection, q to quit, s to search.\n"
	} else {
		s += "\n ↑/↓ to move, space/enter to toggle selection, q to quit, s to search, e to edit base dir.\n"
	}

	s += fmt.Sprintf("%d x %d", m.height, m.width)
	return s
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
