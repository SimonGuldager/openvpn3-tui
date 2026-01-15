package ui

import (
	"fmt"
	"os"
	"strings"

	"openvpn3-tui/internal/config"
	"openvpn3-tui/internal/openvpn"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// View represents the current view/tab
type View int

const (
	ViewProfiles View = iota
	ViewSessions
)

// InputMode represents what input we're collecting
type InputMode int

const (
	InputNone InputMode = iota
	InputProfilePath
	InputProfileName
)

// Model is the main application model
type Model struct {
	// Core state
	config   *config.Config
	client   *openvpn.Client
	sessions []openvpn.Session

	// UI state
	currentView    View
	profileCursor  int
	sessionCursor  int
	profileValid   map[int]bool
	selectedStats  *openvpn.SessionStats
	loading        bool
	loadingMsg     string
	spinner        spinner.Model
	styles         *Styles

	// Input state
	inputMode  InputMode
	textInput  textinput.Model
	newProfile config.Profile
	completer  *PathCompleter

	// Messages
	statusMsg string
	errorMsg  string

	// Dimensions
	width  int
	height int
}

// sessionRefreshMsg is sent when sessions need to be refreshed
type sessionRefreshMsg struct {
	sessions []openvpn.Session
	err      error
}

// statsRefreshMsg is sent when stats are fetched
type statsRefreshMsg struct {
	stats *openvpn.SessionStats
	err   error
}

// connectMsg is sent after a connection attempt
type connectMsg struct {
	err error
}

// disconnectMsg is sent after a disconnect attempt
type disconnectMsg struct {
	err error
}

// NewModel creates a new application model
func NewModel(cfg *config.Config) Model {
	// Load theme and create styles
	theme := LoadTheme()
	styles := NewStyles(theme)

	ti := textinput.New()
	ti.Placeholder = "Enter path to .ovpn file (start with ~ or /)"
	ti.CharLimit = 256
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	return Model{
		config:       cfg,
		client:       openvpn.NewClient(),
		profileValid: cfg.ValidateProfiles(),
		textInput:    ti,
		completer:    NewPathCompleter(),
		spinner:      s,
		styles:       styles,
		loading:      true,
		loadingMsg:   "Fetching sessions...",
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.refreshSessions(), WatchTheme())
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle input mode separately
		if m.inputMode != InputNone {
			return m.handleInputMode(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			if m.currentView == ViewProfiles {
				m.currentView = ViewSessions
			} else {
				m.currentView = ViewProfiles
			}
			m.clearMessages()

		case "up", "k":
			m.moveCursorUp()

		case "down", "j":
			m.moveCursorDown()

		case "enter":
			return m.handleEnter()

		case "a":
			if m.currentView == ViewProfiles {
				return m.startAddProfile()
			}

		case "d", "delete":
			return m.handleDelete()

		case "r":
			m.clearMessages()
			m.loading = true
			m.loadingMsg = "Refreshing sessions..."
			return m, tea.Batch(m.spinner.Tick, m.refreshSessions())

		case "s":
			if m.currentView == ViewSessions && len(m.sessions) > 0 {
				m.loading = true
				m.loadingMsg = "Fetching stats..."
				return m, tea.Batch(m.spinner.Tick, m.fetchStats(m.sessions[m.sessionCursor].Path))
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case sessionRefreshMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch sessions: %v", msg.err)
		} else {
			m.sessions = msg.sessions
			if m.sessionCursor >= len(m.sessions) {
				m.sessionCursor = max(0, len(m.sessions)-1)
			}
		}

	case statsRefreshMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch stats: %v", msg.err)
		} else {
			m.selectedStats = msg.stats
		}

	case connectMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Connection failed: %v", msg.err)
		} else {
			m.statusMsg = "Connected successfully!"
			m.loading = true
			m.loadingMsg = "Refreshing sessions..."
			cmds = append(cmds, m.spinner.Tick, m.refreshSessions())
		}

	case disconnectMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Disconnect failed: %v", msg.err)
		} else {
			m.statusMsg = "Disconnected successfully!"
			m.selectedStats = nil
			m.loading = true
			m.loadingMsg = "Refreshing sessions..."
			cmds = append(cmds, m.spinner.Tick, m.refreshSessions())
		}

	case ThemeChangedMsg:
		// Reload theme and recreate styles
		theme := LoadTheme()
		m.styles = NewStyles(theme)
		m.spinner.Style = m.styles.Spinner
		// Restart the theme watcher
		cmds = append(cmds, WatchTheme())
	}

	return m, tea.Batch(cmds...)
}

// handleInputMode handles key events during input mode
func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputMode = InputNone
		m.newProfile = config.Profile{}
		m.completer.Clear()
		return m, nil

	case "tab":
		// Tab completion - only in path input mode
		if m.inputMode == InputProfilePath && m.completer.HasSuggestions() {
			m.completer.SelectNext()
			if selected := m.completer.GetSelected(); selected != "" {
				m.textInput.SetValue(selected)
				m.textInput.CursorEnd()
				// Update suggestions for the new value
				m.completer.Update(selected)
			}
		}
		return m, nil

	case "shift+tab":
		// Reverse tab completion
		if m.inputMode == InputProfilePath && m.completer.HasSuggestions() {
			m.completer.SelectPrev()
			if selected := m.completer.GetSelected(); selected != "" {
				m.textInput.SetValue(selected)
				m.textInput.CursorEnd()
				m.completer.Update(selected)
			}
		}
		return m, nil

	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value == "" {
			return m, nil
		}

		if m.inputMode == InputProfilePath {
			// Expand ~ before saving
			expandedPath := value
			if strings.HasPrefix(value, "~") {
				if home, err := os.UserHomeDir(); err == nil {
					expandedPath = home + value[1:]
				}
			}
			m.newProfile.Path = expandedPath
			m.inputMode = InputProfileName
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Enter a friendly name"
			m.completer.Clear()
			return m, nil
		}

		if m.inputMode == InputProfileName {
			m.newProfile.Name = value
			m.config.AddProfile(m.newProfile.Name, m.newProfile.Path)
			if err := m.config.Save(); err != nil {
				m.errorMsg = fmt.Sprintf("Failed to save config: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("Added profile: %s", m.newProfile.Name)
			}
			m.profileValid = m.config.ValidateProfiles()
			m.inputMode = InputNone
			m.newProfile = config.Profile{}
			m.completer.Clear()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Update path suggestions after each keystroke (only in path mode)
	if m.inputMode == InputProfilePath {
		m.completer.Update(m.textInput.Value())
	}

	return m, cmd
}

// startAddProfile enters input mode for adding a profile
func (m Model) startAddProfile() (tea.Model, tea.Cmd) {
	m.inputMode = InputProfilePath
	m.textInput.SetValue("")
	m.textInput.Placeholder = "Enter path to .ovpn file"
	m.textInput.Focus()
	m.clearMessages()
	return m, textinput.Blink
}

// handleEnter handles the enter key based on current view
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	m.clearMessages()

	if m.currentView == ViewProfiles {
		if len(m.config.Profiles) == 0 {
			return m, nil
		}
		if !m.profileValid[m.profileCursor] {
			m.errorMsg = "Config file not found"
			return m, nil
		}
		profile := m.config.Profiles[m.profileCursor]

		// Check if already connected
		if m.isProfileConnected(profile.Path) {
			m.errorMsg = fmt.Sprintf("'%s' is already connected", profile.Name)
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Connecting to %s...", profile.Name)
		m.loading = true
		m.loadingMsg = "Connecting..."
		return m, tea.Batch(m.spinner.Tick, m.connect(profile.Path))
	}

	if m.currentView == ViewSessions {
		if len(m.sessions) > 0 {
			m.loading = true
			m.loadingMsg = "Fetching stats..."
			return m, tea.Batch(m.spinner.Tick, m.fetchStats(m.sessions[m.sessionCursor].Path))
		}
	}

	return m, nil
}

// isProfileConnected checks if a profile is already connected
func (m Model) isProfileConnected(profilePath string) bool {
	// Extract filename without extension from profile path
	profileName := profilePath
	if lastSlash := strings.LastIndex(profilePath, "/"); lastSlash != -1 {
		profileName = profilePath[lastSlash+1:]
	}
	profileName = strings.TrimSuffix(profileName, ".ovpn")

	// Check against active sessions
	for _, session := range m.sessions {
		if session.ConfigName == profileName {
			return true
		}
	}
	return false
}

// handleDelete handles deletion based on current view
func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	m.clearMessages()

	if m.currentView == ViewProfiles {
		if len(m.config.Profiles) > 0 {
			name := m.config.Profiles[m.profileCursor].Name
			m.config.RemoveProfile(m.profileCursor)
			if err := m.config.Save(); err != nil {
				m.errorMsg = fmt.Sprintf("Failed to save config: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("Removed profile: %s", name)
			}
			m.profileValid = m.config.ValidateProfiles()
			if m.profileCursor >= len(m.config.Profiles) {
				m.profileCursor = max(0, len(m.config.Profiles)-1)
			}
		}
		return m, nil
	}

	if m.currentView == ViewSessions {
		if len(m.sessions) > 0 {
			session := m.sessions[m.sessionCursor]
			m.statusMsg = "Disconnecting..."
			return m, m.disconnect(session.Path)
		}
	}

	return m, nil
}

// moveCursorUp moves the cursor up in the current list
func (m *Model) moveCursorUp() {
	if m.currentView == ViewProfiles {
		if m.profileCursor > 0 {
			m.profileCursor--
		}
	} else {
		if m.sessionCursor > 0 {
			m.sessionCursor--
		}
	}
	m.selectedStats = nil
}

// moveCursorDown moves the cursor down in the current list
func (m *Model) moveCursorDown() {
	if m.currentView == ViewProfiles {
		if m.profileCursor < len(m.config.Profiles)-1 {
			m.profileCursor++
		}
	} else {
		if m.sessionCursor < len(m.sessions)-1 {
			m.sessionCursor++
		}
	}
	m.selectedStats = nil
}

// clearMessages clears status and error messages
func (m *Model) clearMessages() {
	m.statusMsg = ""
	m.errorMsg = ""
}

// Commands

func (m Model) refreshSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.client.ListSessions()
		return sessionRefreshMsg{sessions: sessions, err: err}
	}
}

func (m Model) fetchStats(path string) tea.Cmd {
	return func() tea.Msg {
		stats, err := m.client.GetSessionStats(path)
		return statsRefreshMsg{stats: stats, err: err}
	}
}

func (m Model) connect(configPath string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.Connect(configPath)
		return connectMsg{err: err}
	}
}

func (m Model) disconnect(sessionPath string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.Disconnect(sessionPath)
		return disconnectMsg{err: err}
	}
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render("OpenVPN3 TUI"))
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Input mode
	if m.inputMode != InputNone {
		b.WriteString(m.renderInputMode())
		return b.String()
	}

	// Main content based on current view
	if m.currentView == ViewProfiles {
		b.WriteString(m.renderProfiles())
	} else {
		b.WriteString(m.renderSessions())
	}

	// Loading indicator
	if m.loading {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), m.loadingMsg))
	}

	// Messages
	if m.errorMsg != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Error.Render(m.errorMsg))
	}
	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Success.Render(m.statusMsg))
	}

	// Help
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderTabs() string {
	var tabs []string

	if m.currentView == ViewProfiles {
		tabs = append(tabs, m.styles.ActiveTab.Render("Profiles"))
		tabs = append(tabs, m.styles.InactiveTab.Render("Sessions"))
	} else {
		tabs = append(tabs, m.styles.InactiveTab.Render("Profiles"))
		tabs = append(tabs, m.styles.ActiveTab.Render("Sessions"))
	}

	return strings.Join(tabs, "  ")
}

func (m Model) renderInputMode() string {
	var b strings.Builder

	title := "Add Profile - Enter Path"
	if m.inputMode == InputProfileName {
		title = "Add Profile - Enter Name"
	}

	b.WriteString(m.styles.Subtitle.Render(title))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n")

	// Show path suggestions
	if m.inputMode == InputProfilePath && m.completer.HasSuggestions() {
		b.WriteString("\n")
		suggestions := m.completer.Suggestions()
		selectedIdx := m.completer.SelectedIndex()

		for i, suggestion := range suggestions {
			displayPath := CompactPath(suggestion)
			if i == selectedIdx {
				b.WriteString(m.styles.SuggestionSelected.Render("  > " + displayPath))
			} else {
				b.WriteString(m.styles.Suggestion.Render("    " + displayPath))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.inputMode == InputProfilePath {
		b.WriteString(m.styles.Help.Render("tab: complete • enter: confirm • esc: cancel"))
	} else {
		b.WriteString(m.styles.Help.Render("enter: confirm • esc: cancel"))
	}

	return m.styles.Box.Render(b.String())
}

func (m Model) renderProfiles() string {
	var b strings.Builder

	if len(m.config.Profiles) == 0 {
		b.WriteString(m.styles.Subtitle.Render("No profiles configured"))
		b.WriteString("\n")
		b.WriteString("Press 'a' to add a profile")
		return b.String()
	}

	for i, profile := range m.config.Profiles {
		cursor := "  "
		if i == m.profileCursor {
			cursor = "> "
		}

		isConnected := m.isProfileConnected(profile.Path)
		line := fmt.Sprintf("%s%s", cursor, profile.Name)

		if i == m.profileCursor {
			b.WriteString(m.styles.Selected.Render(line))
			if isConnected {
				b.WriteString(" ")
				b.WriteString(m.styles.Connected.Render("[connected]"))
			}
		} else if !m.profileValid[i] {
			b.WriteString(m.styles.Invalid.Render(line + " (file not found)"))
		} else if isConnected {
			b.WriteString(m.styles.Normal.Render(line))
			b.WriteString(" ")
			b.WriteString(m.styles.Connected.Render("[connected]"))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderSessions() string {
	var b strings.Builder

	if len(m.sessions) == 0 {
		b.WriteString(m.styles.Subtitle.Render("No active sessions"))
		return b.String()
	}

	for i, session := range m.sessions {
		cursor := "  "
		if i == m.sessionCursor {
			cursor = "> "
		}

		status := session.Status
		var statusStyled string
		switch {
		case strings.Contains(strings.ToLower(status), "connected"):
			statusStyled = m.styles.Connected.Render(status)
		case strings.Contains(strings.ToLower(status), "paused"):
			statusStyled = m.styles.Paused.Render(status)
		default:
			statusStyled = m.styles.Disconnected.Render(status)
		}

		line := fmt.Sprintf("%s%s [%s]", cursor, session.ConfigName, statusStyled)

		if i == m.sessionCursor {
			b.WriteString(m.styles.Selected.Render(fmt.Sprintf("%s%s", cursor, session.ConfigName)))
			b.WriteString(fmt.Sprintf(" [%s]", statusStyled))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	// Show stats if available
	if m.selectedStats != nil && m.sessionCursor < len(m.sessions) {
		b.WriteString(m.renderStats())
	}

	return b.String()
}

func (m Model) renderStats() string {
	stats := m.selectedStats
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Tunnel IP:   %s\n", stats.TunnelIP))
	if stats.TunnelIPv6 != "" {
		sb.WriteString(fmt.Sprintf("Tunnel IPv6: %s\n", stats.TunnelIPv6))
	}
	sb.WriteString(fmt.Sprintf("Bytes In:    %s\n", stats.BytesIn))
	sb.WriteString(fmt.Sprintf("Bytes Out:   %s\n", stats.BytesOut))
	sb.WriteString(fmt.Sprintf("Packets In:  %s\n", stats.PacketsIn))
	sb.WriteString(fmt.Sprintf("Packets Out: %s\n", stats.PacketsOut))
	if stats.Connected != "" {
		sb.WriteString(fmt.Sprintf("Connected:   %s", stats.Connected))
	}

	return m.styles.StatsBox.Render(sb.String())
}

func (m Model) renderHelp() string {
	var help string
	if m.currentView == ViewProfiles {
		help = "tab: switch view • j/k: navigate • enter: connect • a: add • d: delete • r: refresh • q: quit"
	} else {
		help = "tab: switch view • j/k: navigate • enter/s: stats • d: disconnect • r: refresh • q: quit"
	}
	return m.styles.Help.Render(help)
}
