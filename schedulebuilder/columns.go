package main

// Column represents a selectable list of items with cursor navigation
type Column struct {
	items    []string
	cursor   int
	selected map[int]struct{}
}

func newColumn() Column {
	return Column{
		items:    []string{},
		cursor:   0,
		selected: make(map[int]struct{}),
	}
}

func (c *Column) setItems(items []string) {
	c.items = items
	if c.cursor >= len(items) && len(items) > 0 {
		c.cursor = len(items) - 1
	}
	if c.cursor < 0 && len(items) > 0 {
		c.cursor = 0
	}
}

func (c *Column) len() int {
	if len(c.items) > 0 {
		return len(c.items)
	}
	return 1
}

func (c *Column) moveCursor(delta int) {
	c.cursor += delta
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= c.len() {
		c.cursor = c.len() - 1
	}
}

func (c *Column) toggleSelection() {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return
	}
	if _, ok := c.selected[c.cursor]; ok {
		delete(c.selected, c.cursor)
	} else {
		c.selected[c.cursor] = struct{}{}
	}
}

func (c *Column) isSelected(idx int) bool {
	_, ok := c.selected[idx]
	return ok
}

func (c *Column) getSelected() []string {
	result := []string{}
	for idx := range c.selected {
		if idx < len(c.items) {
			result = append(result, c.items[idx])
		}
	}
	return result
}
