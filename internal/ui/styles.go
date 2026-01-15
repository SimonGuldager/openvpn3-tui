package ui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the application styles
type Styles struct {
	Title               lipgloss.Style
	Subtitle            lipgloss.Style
	Selected            lipgloss.Style
	Normal              lipgloss.Style
	Connected           lipgloss.Style
	Disconnected        lipgloss.Style
	Paused              lipgloss.Style
	Box                 lipgloss.Style
	StatsBox            lipgloss.Style
	Help                lipgloss.Style
	Error               lipgloss.Style
	Success             lipgloss.Style
	Invalid             lipgloss.Style
	ActiveTab           lipgloss.Style
	InactiveTab         lipgloss.Style
	Suggestion          lipgloss.Style
	SuggestionSelected  lipgloss.Style
	Spinner             lipgloss.Style
}

// NewStyles creates styles from a theme
func NewStyles(t *Theme) *Styles {
	return &Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.Muted).
			MarginBottom(1),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.SelectionForeground).
			Background(t.Accent).
			Padding(0, 1),

		Normal: lipgloss.NewStyle().
			Padding(0, 1),

		Connected: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		Disconnected: lipgloss.NewStyle().
			Foreground(t.Error),

		Paused: lipgloss.NewStyle().
			Foreground(t.Warning),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Accent).
			Padding(1, 2),

		StatsBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Success).
			Padding(1, 2).
			MarginTop(1),

		Help: lipgloss.NewStyle().
			Foreground(t.Muted).
			MarginTop(1),

		Error: lipgloss.NewStyle().
			Foreground(t.Error).
			Bold(true),

		Success: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		Invalid: lipgloss.NewStyle().
			Foreground(t.Error).
			Strikethrough(true),

		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Accent).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(t.Accent).
			Padding(0, 2),

		InactiveTab: lipgloss.NewStyle().
			Foreground(t.Muted).
			Padding(0, 2),

		Suggestion: lipgloss.NewStyle().
			Foreground(t.Muted),

		SuggestionSelected: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		Spinner: lipgloss.NewStyle().
			Foreground(t.Accent),
	}
}

// DefaultStyles creates styles with the default theme
func DefaultStyles() *Styles {
	return NewStyles(DefaultTheme())
}
