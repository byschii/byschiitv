package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

type model struct {
	// screen 0 = ask base dir, 1 = main panel
	screen int

	// text input for base dir prompt
	ti      textinput.Model
	baseDir string
	// error message to show on the base dir prompt if validation fails
	errMsg string

	// main panel state (existing)
	// two columns: 0 = scanned/search (left), 1 = planned/built so far (right)
	planned      []string
	scanned      []string
	cursor       int // row index within active column
	activeColumn int // 0=scanned,1=planned
	selected     map[string]struct{}

	// prepared for next steps: search query / rapid ops selections
	searchQuery string
	// search mode state
	searchMode    bool
	searchInput   textinput.Model
	searchHistory []string
	historyIdx    int
	// live results while searching (ordered by Levenshtein distance)
	searchResults []string
}

func initialModel() model {
	ti := textinput.New()
	// default value: HOST_MEDIA_PATH or fallback
	defaultPath := os.Getenv("HOST_MEDIA_PATH")
	if defaultPath == "" {
		defaultPath = "./byschiitv/media"
	}
	ti.Placeholder = defaultPath
	ti.SetValue(defaultPath)
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 60

	// prepare an empty planned list (built so far) by default
	planned := []string{}

	return model{
		screen:       0,
		ti:           ti,
		baseDir:      defaultPath,
		errMsg:       "",
		planned:      planned,
		scanned:      []string{},
		cursor:       0,
		activeColumn: 0, // start focused on scanned/search column
		selected:     make(map[string]struct{}),
		searchMode:   false,
		searchInput: func() textinput.Model {
			t := textinput.New()
			t.Placeholder = "search..."
			t.CharLimit = 256
			t.Width = 60
			return t
		}(),
		searchHistory: []string{},
		historyIdx:    -1,
		searchResults: []string{},
	}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// screen 0: base dir prompt using textinput
	if m.screen == 0 {
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// accept value but validate path exists and is a directory before switching
				m.baseDir = m.ti.Value()
				if info, err := os.Stat(m.baseDir); err != nil {
					m.errMsg = fmt.Sprintf("path error: %v", err)
					// stay on prompt
				} else if !info.IsDir() {
					m.errMsg = "path exists but is not a directory"
				} else {
					// clear any prior error and proceed to main panel
					m.errMsg = ""
					m.screen = 1
					// scan the base directory for media files and populate center column
					m.scanned = scanMedia(m.baseDir)
					// reset cursor to top of scanned column (first/left column)
					m.activeColumn = 0
					m.cursor = 0
				}
			case "ctrl+c":
				return m, tea.Quit
			}
		}
		return m, cmd
	}

	// screen 1: multi-column selection UI
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// if we're in search mode, route keys to the search input and history
		if m.searchMode {
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			// navigation through history using up/down while focused on the input
			switch msg.String() {
			case "enter":
				// commit the search: save to history, generate ordered results, exit search mode
				q := strings.TrimSpace(m.searchInput.Value())
				if q != "" {
					// avoid consecutive duplicates
					if len(m.searchHistory) == 0 || m.searchHistory[len(m.searchHistory)-1] != q {
						m.searchHistory = append(m.searchHistory, q)
					}
					m.historyIdx = len(m.searchHistory)
				}
				m.searchQuery = q
				m.searchResults = sortByLevenshtein(m.scanned, q)
				// apply results to scanned view and return to list mode
				if len(m.searchResults) > 0 {
					m.scanned = m.searchResults
				}
				m.searchMode = false
				m.cursor = 0
				return m, cmd
			case "esc":
				// cancel search
				m.searchMode = false
				m.searchResults = nil
				return m, cmd
			case "up", "k":
				if len(m.searchHistory) > 0 {
					if m.historyIdx <= 0 {
						m.historyIdx = len(m.searchHistory) - 1
					} else {
						m.historyIdx--
					}
					m.searchInput.SetValue(m.searchHistory[m.historyIdx])
				}
				return m, cmd
			case "down", "j":
				if len(m.searchHistory) > 0 {
					if m.historyIdx >= len(m.searchHistory)-1 {
						m.historyIdx = 0
					} else {
						m.historyIdx++
					}
					m.searchInput.SetValue(m.searchHistory[m.historyIdx])
				}
				return m, cmd
			default:
				// live-update search results as the user types
				q := strings.TrimSpace(m.searchInput.Value())
				if q == "" {
					m.searchResults = m.scanned
				} else {
					m.searchResults = sortByLevenshtein(m.scanned, q)
				}
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			if m.activeColumn > 0 {
				m.activeColumn--
				// clamp cursor to new column length
				max := columnLen(m, m.activeColumn)
				if m.cursor >= max {
					m.cursor = max - 1
					if m.cursor < 0 {
						m.cursor = 0
					}
				}
			}
		case "right", "l":
			// intentionally disabled: keep focus on the left (scanned/search) column only
			// no-op so the cursor does not move to the built/planned column
		case "s":
			// enter search mode
			m.searchMode = true
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			m.historyIdx = len(m.searchHistory)
			// prepare initial results
			m.searchResults = m.scanned
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < columnLen(m, m.activeColumn)-1 {
				m.cursor++
			}
		case "enter", " ":
			idx := m.cursor
			key := fmt.Sprintf("%d:%d", m.activeColumn, idx)
			if _, ok := m.selected[key]; ok {
				delete(m.selected, key)
			} else {
				m.selected[key] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.screen == 0 {
		s := "Enter base directory for videos (press Enter to continue)\n\n"
		s += m.ti.View() + "\n\n"
		s += fmt.Sprintf("Detected default (from HOST_MEDIA_PATH or fallback): %s\n", m.ti.Placeholder)
		if m.errMsg != "" {
			s += "\nError: " + m.errMsg + "\n"
		}
		s += "\nPress ctrl+c to quit.\n"
		return s
	}

	s := fmt.Sprintf("Schedule Builder — base dir: %s\nselect items (space/enter) and press q to quit\n\n", m.baseDir)
	// two-column layout: scanned/search (left) | planned/built (right)
	leftTitle := "Search"
	rightTitle := "Built so far"

	// column widths: compute main (left) width from COLUMNS env (if present)
	// and reserve a reasonable width for the built/right column.
	totalWidth := 120
	if c := os.Getenv("COLUMNS"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v > 40 {
			totalWidth = v
		}
	}
	// reserve for built column
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

	// helper to truncate strings with rune-safety and '...'
	truncate := func(s string, max int) string {
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
	s += header

	// if in search mode show the text input above the left column
	if m.searchMode {
		s += m.searchInput.View() + "\n"
		s += "(type to narrow results, Enter to apply, Esc to cancel)\n\n"
	}

	// decide which left-column items to render: either scanned or live searchResults
	leftItems := m.scanned
	if m.searchMode && len(m.searchResults) > 0 {
		leftItems = m.searchResults
	}

	// determine number of rows to print
	rows := len(leftItems)
	if rows == 0 {
		rows = 1
	}
	if rl := columnLen(m, 1); rl > rows {
		rows = rl
	}

	for i := 0; i < rows; i++ {
		// left (search/scanned)
		left := ""
		if i < len(leftItems) {
			left = leftItems[i]
		}
		// right (planned)
		right := ""
		if i < len(m.planned) {
			right = m.planned[i]
		}

		// cursor markers per column
		lcur, rcur := " ", " "
		if m.activeColumn == 0 && m.cursor == i {
			lcur = ">"
		}
		if m.activeColumn == 1 && m.cursor == i {
			rcur = ">"
		}

		// checked marker for left column only
		lchk := " "
		if _, ok := m.selected[fmt.Sprintf("0:%d", i)]; ok {
			lchk = "x"
		}

		// compute content widths and truncate if necessary
		leftWidth := lw - 6 // space for cursor and [x]
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

	s += "\n ↑/↓ to move, space/enter to toggle selection, q to quit, s to search.\n"
	return s
}

// columnLen returns the length of the given column for rendering/navigation
func columnLen(m model, col int) int {
	switch col {
	case 0:
		if l := len(m.scanned); l > 0 {
			return l
		}
		return 1
	case 1:
		if l := len(m.planned); l > 0 {
			return l
		}
		return 1
	default:
		return 0
	}
}

// scanMedia walks the provided directory and returns a list of media files (relative paths)
func scanMedia(root string) []string {
	var files []string
	extensions := map[string]struct{}{
		".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {}, ".flv": {}, ".wmv": {},
		".mpg": {}, ".mpeg": {}, ".webm": {}, ".m4v": {}, ".ts": {},
	}

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// skip unreadable files/dirs
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := extensions[ext]; ok {
			// record relative path if possible
			rel := path
			if r, err := filepath.Rel(root, path); err == nil {
				rel = r
			} else {
				rel = strings.Replace(path, root+string(os.PathSeparator), "", 1)
			}
			files = append(files, rel)
		}
		return nil
	})
	return files
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	// simple iterative implementation
	la := len(a)
	lb := len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		dp[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= lb; j++ {
			cur := dp[j]
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			// substitution
			v := prev + cost
			// insertion
			if dp[j-1]+1 < v {
				v = dp[j-1] + 1
			}
			// deletion
			if dp[j]+1 < v {
				v = dp[j] + 1
			}
			prev = cur
			dp[j] = v
		}
	}
	return dp[lb]
}

// sortByLevenshtein returns a new slice of inputs sorted by Levenshtein distance to q (ascending).
func sortByLevenshtein(inputs []string, q string) []string {
	if q == "" {
		out := make([]string, len(inputs))
		copy(out, inputs)
		return out
	}
	type pair struct {
		s string
		d int
	}
	ps := make([]pair, 0, len(inputs))
	for _, s := range inputs {
		d := levenshtein(strings.ToLower(s), strings.ToLower(q))
		ps = append(ps, pair{s: s, d: d})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].d == ps[j].d {
			return ps[i].s < ps[j].s
		}
		return ps[i].d < ps[j].d
	})
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.s
	}
	return out
}

func main() {
	// load .env (optional). If .env is missing we continue; env vars from the
	// environment are still used (e.g. HOST_MEDIA_PATH).
	if err := godotenv.Load("../.env"); err != nil {
		// Not fatal: just continue. Uncomment next line to debug missing file.
		// fmt.Fprintln(os.Stderr, "No .env file loaded:", err)
	}

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
