package main

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchBox manages search input, history, and live filtering
type SearchBox struct {
	input      textinput.Model
	history    []string
	historyIdx int
	active     bool
}

func newSearchBox() SearchBox {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 256
	ti.Width = 60
	return SearchBox{
		input:      ti,
		history:    []string{},
		historyIdx: -1,
		active:     false,
	}
}

func (s *SearchBox) activate() {
	s.active = true
	s.input.Focus()
	s.input.SetValue("")
	s.historyIdx = len(s.history)
}

func (s *SearchBox) deactivate() {
	s.active = false
	s.input.Blur()
}

func (s *SearchBox) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

func (s *SearchBox) value() string {
	return strings.TrimSpace(s.input.Value())
}

func (s *SearchBox) commit() {
	q := s.value()
	if q != "" {
		if len(s.history) == 0 || s.history[len(s.history)-1] != q {
			s.history = append(s.history, q)
		}
		s.historyIdx = len(s.history)
	}
}

func (s *SearchBox) prevHistory() {
	if len(s.history) > 0 {
		if s.historyIdx <= 0 {
			s.historyIdx = len(s.history) - 1
		} else {
			s.historyIdx--
		}
		s.input.SetValue(s.history[s.historyIdx])
	}
}

func (s *SearchBox) nextHistory() {
	if len(s.history) > 0 {
		if s.historyIdx >= len(s.history)-1 {
			s.historyIdx = 0
		} else {
			s.historyIdx++
		}
		s.input.SetValue(s.history[s.historyIdx])
	}
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
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
			v := prev + cost
			if dp[j-1]+1 < v {
				v = dp[j-1] + 1
			}
			if dp[j]+1 < v {
				v = dp[j] + 1
			}
			prev = cur
			dp[j] = v
		}
	}
	return dp[lb]
}

// sortByLevenshtein returns a new slice of inputs sorted by Levenshtein distance to query (ascending).
func sortByLevenshtein(inputs []string, query string) []string {
	if query == "" {
		out := make([]string, len(inputs))
		copy(out, inputs)
		return out
	}
	type pair struct {
		s string
		d int
	}
	ps := make([]pair, 0, len(inputs))
	qlower := strings.ToLower(query)
	for _, s := range inputs {
		d := levenshtein(stripstring(s), qlower)
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

// ngrams returns a map of character n-gram -> count using runes.
func ngrams(s string, n int) map[string]int {
	m := make(map[string]int)
	r := []rune(strings.ToLower(s))
	if n <= 0 {
		return m
	}
	if len(r) < n {
		if len(r) > 0 {
			m[string(r)]++
		}
		return m
	}
	for i := 0; i <= len(r)-n; i++ {
		m[string(r[i:i+n])]++
	}
	return m
}

// CosineNGram computes cosine similarity between two strings using character n-grams.
// Returns value in [0,1], where 1 means identical n-gram vectors.
func CosineNGram(a, b string, n int) float64 {
	if a == b {
		return 1.0
	}
	ma := ngrams(a, n)
	mb := ngrams(b, n)
	var dot float64
	var na2 float64
	var nb2 float64
	for k, va := range ma {
		vb := mb[k]
		dot += float64(va * vb)
		na2 += float64(va * va)
	}
	for _, vb := range mb {
		nb2 += float64(vb * vb)
	}
	if na2 == 0 || nb2 == 0 {
		return 0
	}
	return dot / math.Sqrt(na2*nb2)
}

// tokenize builds a set of tokens from the input string. Tokens are sequences of letters or numbers.
func tokenize(s string) map[string]struct{} {
	out := make(map[string]struct{})
	lower := strings.ToLower(s)
	f := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}
	for _, t := range strings.FieldsFunc(lower, f) {
		if t == "" {
			continue
		}
		out[t] = struct{}{}
	}
	return out
}

// JaccardTokenSet computes Jaccard similarity between token sets of two strings.
// Tokens are extracted by splitting on non-letter/non-digit characters. Result in [0,1].
func JaccardTokenSet(a, b string) float64 {
	sa := tokenize(a)
	sb := tokenize(b)
	if len(sa) == 0 && len(sb) == 0 {
		return 1.0
	}
	if len(sa) == 0 || len(sb) == 0 {
		return 0.0
	}
	inter := 0
	for k := range sa {
		if _, ok := sb[k]; ok {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0.0
	}
	return float64(inter) / float64(union)
}

// sortByCosine sorts inputs by cosine n-gram similarity to query (descending).
func sortByCosine(inputs []string, query string, n int) []string {
	if query == "" {
		out := make([]string, len(inputs))
		copy(out, inputs)
		return out
	}
	type pair struct {
		s string
		v float64
	}
	ps := make([]pair, 0, len(inputs))
	for _, s := range inputs {
		v := CosineNGram(stripstring(s), query, n)
		ps = append(ps, pair{s: s, v: v})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].v == ps[j].v {
			return ps[i].s < ps[j].s
		}
		return ps[i].v > ps[j].v
	})
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.s
	}
	return out
}

// sortByJaccard sorts inputs by Jaccard token-set similarity to query (descending).
func sortByJaccard(inputs []string, query string) []string {
	if query == "" {
		out := make([]string, len(inputs))
		copy(out, inputs)
		return out
	}
	type pair struct {
		s string
		v float64
	}
	ps := make([]pair, 0, len(inputs))
	for _, s := range inputs {
		v := JaccardTokenSet(stripstring(s), query)
		ps = append(ps, pair{s: s, v: v})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].v == ps[j].v {
			return ps[i].s < ps[j].s
		}
		return ps[i].v > ps[j].v
	})
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.s
	}
	return out
}

func stripstring(s string) string {
	// remove spaces and punctuation, convert to lower case
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
