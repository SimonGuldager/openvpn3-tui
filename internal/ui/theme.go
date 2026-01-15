package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// Theme holds the color scheme
type Theme struct {
	Accent              lipgloss.Color
	Foreground          lipgloss.Color
	Background          lipgloss.Color
	SelectionForeground lipgloss.Color
	SelectionBackground lipgloss.Color

	// Semantic colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Muted   lipgloss.Color
}

// DefaultTheme returns a sensible default color scheme
func DefaultTheme() *Theme {
	return &Theme{
		Accent:              lipgloss.Color("#7C3AED"),
		Foreground:          lipgloss.Color("#d8dee9"),
		Background:          lipgloss.Color("#2e3440"),
		SelectionForeground: lipgloss.Color("#FFFFFF"),
		SelectionBackground: lipgloss.Color("#7C3AED"),
		Success:             lipgloss.Color("#a3be8c"),
		Warning:             lipgloss.Color("#ebcb8b"),
		Error:               lipgloss.Color("#bf616a"),
		Muted:               lipgloss.Color("#6B7280"),
	}
}

// LoadTheme loads colors from ~/.config/openvpn3-tui/theme.toml or returns defaults
func LoadTheme() *Theme {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultTheme()
	}

	themePath := filepath.Join(home, ".config", "openvpn3-tui", "theme.toml")
	return loadThemeFromFile(themePath)
}

// loadThemeFromFile parses a theme.toml file
func loadThemeFromFile(path string) *Theme {
	file, err := os.Open(path)
	if err != nil {
		return DefaultTheme()
	}
	defer file.Close()

	theme := DefaultTheme()
	colors := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		colors[key] = value
	}

	// Map parsed colors to theme
	if v, ok := colors["accent"]; ok {
		theme.Accent = lipgloss.Color(v)
	}
	if v, ok := colors["foreground"]; ok {
		theme.Foreground = lipgloss.Color(v)
	}
	if v, ok := colors["background"]; ok {
		theme.Background = lipgloss.Color(v)
	}
	if v, ok := colors["selection_foreground"]; ok {
		theme.SelectionForeground = lipgloss.Color(v)
	}
	if v, ok := colors["selection_background"]; ok {
		theme.SelectionBackground = lipgloss.Color(v)
	}
	if v, ok := colors["success"]; ok {
		theme.Success = lipgloss.Color(v)
	}
	if v, ok := colors["warning"]; ok {
		theme.Warning = lipgloss.Color(v)
	}
	if v, ok := colors["error"]; ok {
		theme.Error = lipgloss.Color(v)
	}
	if v, ok := colors["muted"]; ok {
		theme.Muted = lipgloss.Color(v)
	}

	return theme
}

// ThemeChangedMsg is sent when the theme file changes
type ThemeChangedMsg struct{}

// ThemePath returns the path to the theme file
func ThemePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "openvpn3-tui", "theme.toml")
}

// WatchTheme starts watching for theme changes
// Watches the omarchy current theme directory since it gets replaced on theme switch
func WatchTheme() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}

		// Watch the parent of the theme directory - omarchy swaps the entire "theme" dir
		// So we watch "current" for the "theme" directory being recreated
		watchDir := filepath.Join(home, ".config", "omarchy", "current")

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}

		if err := watcher.Add(watchDir); err != nil {
			watcher.Close()
			return nil
		}

		// Wait for theme directory to be created (happens after mv in theme-set)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					watcher.Close()
					return nil
				}
				// Detect when theme directory is created or renamed into place
				if filepath.Base(event.Name) == "theme" {
					if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
						watcher.Close()
						return ThemeChangedMsg{}
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					watcher.Close()
					return nil
				}
			}
		}
	}
}
