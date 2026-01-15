package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PathCompleter provides filesystem path completion
type PathCompleter struct {
	suggestions    []string
	selectedIndex  int
	maxSuggestions int
}

// NewPathCompleter creates a new path completer
func NewPathCompleter() *PathCompleter {
	return &PathCompleter{
		maxSuggestions: 5,
		selectedIndex:  -1,
	}
}

// Update refreshes suggestions based on the current input
func (c *PathCompleter) Update(input string) {
	c.suggestions = c.getSuggestions(input)
	c.selectedIndex = -1
}

// getSuggestions returns matching paths for the given input
func (c *PathCompleter) getSuggestions(input string) []string {
	if input == "" {
		return nil
	}

	// Expand ~ to home directory
	if strings.HasPrefix(input, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			input = home + input[1:]
		}
	}

	// Get directory and prefix to match
	dir := filepath.Dir(input)
	prefix := filepath.Base(input)

	// If input ends with /, we're looking inside that directory
	if strings.HasSuffix(input, "/") {
		dir = input
		prefix = ""
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		// Try parent directory
		dir = filepath.Dir(dir)
		if _, err := os.Stat(dir); err != nil {
			return nil
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless prefix starts with .
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}

		// Check if name matches prefix (case-insensitive)
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// For directories, add trailing slash
		if entry.IsDir() {
			fullPath += "/"
			matches = append(matches, fullPath)
		} else if strings.HasSuffix(strings.ToLower(name), ".ovpn") {
			// Only show .ovpn files
			matches = append(matches, fullPath)
		}
	}

	// Sort: directories first, then by name
	sort.Slice(matches, func(i, j int) bool {
		iDir := strings.HasSuffix(matches[i], "/")
		jDir := strings.HasSuffix(matches[j], "/")
		if iDir != jDir {
			return iDir // directories first
		}
		return strings.ToLower(matches[i]) < strings.ToLower(matches[j])
	})

	// Limit results
	if len(matches) > c.maxSuggestions {
		matches = matches[:c.maxSuggestions]
	}

	return matches
}

// Suggestions returns the current suggestions
func (c *PathCompleter) Suggestions() []string {
	return c.suggestions
}

// HasSuggestions returns true if there are suggestions available
func (c *PathCompleter) HasSuggestions() bool {
	return len(c.suggestions) > 0
}

// SelectedIndex returns the currently selected suggestion index
func (c *PathCompleter) SelectedIndex() int {
	return c.selectedIndex
}

// SelectNext moves selection to the next suggestion
func (c *PathCompleter) SelectNext() {
	if len(c.suggestions) == 0 {
		return
	}
	c.selectedIndex++
	if c.selectedIndex >= len(c.suggestions) {
		c.selectedIndex = 0
	}
}

// SelectPrev moves selection to the previous suggestion
func (c *PathCompleter) SelectPrev() {
	if len(c.suggestions) == 0 {
		return
	}
	c.selectedIndex--
	if c.selectedIndex < 0 {
		c.selectedIndex = len(c.suggestions) - 1
	}
}

// GetSelected returns the currently selected suggestion, or empty string if none
func (c *PathCompleter) GetSelected() string {
	if c.selectedIndex >= 0 && c.selectedIndex < len(c.suggestions) {
		return c.suggestions[c.selectedIndex]
	}
	return ""
}

// Clear resets the completer state
func (c *PathCompleter) Clear() {
	c.suggestions = nil
	c.selectedIndex = -1
}

// CompactPath shortens a path for display by replacing home dir with ~
func CompactPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
