package openvpn

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Session represents an active OpenVPN3 session
type Session struct {
	Path        string
	ConfigName  string
	Created     string
	Owner       string
	Status      string
	Device      string
	ConnectedTo string
}

// SessionStats holds statistics for a session
type SessionStats struct {
	BytesIn      string
	BytesOut     string
	PacketsIn    string
	PacketsOut   string
	TunnelIP     string
	TunnelIPv6   string
	Connected    string
}

// Client wraps the openvpn3 CLI commands
type Client struct{}

// NewClient creates a new OpenVPN3 client wrapper
func NewClient() *Client {
	return &Client{}
}

// ListSessions returns all active VPN sessions
func (c *Client) ListSessions() ([]Session, error) {
	cmd := exec.Command("openvpn3", "sessions-list")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseSessionsList(output), nil
}

// parseSessionsList parses the output of 'openvpn3 sessions-list'
func parseSessionsList(output []byte) []Session {
	var sessions []Session
	var current *Session

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Skip separator lines
		if strings.HasPrefix(strings.TrimSpace(line), "---") {
			continue
		}

		// Parse key-value pairs - handle multiple fields on same line
		// Format: "     Key: Value                    Key2: Value2"
		if strings.Contains(line, "Path:") {
			if current != nil {
				sessions = append(sessions, *current)
			}
			current = &Session{}
			current.Path = extractValue(line, "Path:")
		} else if current != nil {
			if strings.Contains(line, "Created:") {
				current.Created = extractValue(line, "Created:")
				// Also check for PID on same line (we don't need it but it's there)
			}
			if strings.Contains(line, "Owner:") {
				current.Owner = extractValue(line, "Owner:")
			}
			if strings.Contains(line, "Device:") {
				current.Device = extractValue(line, "Device:")
			}
			if strings.Contains(line, "Config name:") {
				configName := extractValue(line, "Config name:")
				// Remove "(Config not available)" suffix if present
				if idx := strings.Index(configName, "(Config not available)"); idx != -1 {
					configName = strings.TrimSpace(configName[:idx])
				}
				// Extract just the filename from the path
				if lastSlash := strings.LastIndex(configName, "/"); lastSlash != -1 {
					configName = configName[lastSlash+1:]
				}
				// Remove .ovpn extension for cleaner display
				configName = strings.TrimSuffix(configName, ".ovpn")
				current.ConfigName = configName
			}
			if strings.Contains(line, "Connected to:") {
				current.ConnectedTo = extractValue(line, "Connected to:")
			}
			if strings.Contains(line, "Status:") {
				current.Status = extractValue(line, "Status:")
			}
		}
	}

	if current != nil {
		sessions = append(sessions, *current)
	}

	return sessions
}

// extractValue extracts the value after a key from a line
// Handles lines with multiple key:value pairs
func extractValue(line, key string) string {
	idx := strings.Index(line, key)
	if idx == -1 {
		return ""
	}

	// Get everything after the key
	rest := line[idx+len(key):]

	// Find where this value ends (either at next key or end of line)
	// Look for pattern of multiple spaces followed by a capital letter and colon
	value := rest
	for i := 0; i < len(rest)-1; i++ {
		if rest[i] == ' ' && i > 0 {
			// Check if we're at a new field (spaces followed by Key:)
			trimmed := strings.TrimLeft(rest[i:], " ")
			if len(trimmed) > 0 && trimmed[0] >= 'A' && trimmed[0] <= 'Z' {
				if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 && colonIdx < 20 {
					value = rest[:i]
					break
				}
			}
		}
	}

	return strings.TrimSpace(value)
}

// GetSessionStats returns statistics for a given session path
func (c *Client) GetSessionStats(sessionPath string) (*SessionStats, error) {
	cmd := exec.Command("openvpn3", "session-stats", "--path", sessionPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseSessionStats(output), nil
}

// parseSessionStats parses the output of 'openvpn3 session-stats'
func parseSessionStats(output []byte) *SessionStats {
	stats := &SessionStats{}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Format: "     BYTES_IN....................5942"
		// Remove dots and split on remaining whitespace
		line = strings.ReplaceAll(line, ".", " ")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := fields[0]
		value := fields[len(fields)-1]

		switch key {
		case "BYTES_IN":
			stats.BytesIn = formatBytes(value)
		case "BYTES_OUT":
			stats.BytesOut = formatBytes(value)
		case "PACKETS_IN":
			stats.PacketsIn = value
		case "PACKETS_OUT":
			stats.PacketsOut = value
		case "TUN_BYTES_IN":
			stats.TunnelIP = formatBytes(value) + " (TUN in)"
		case "TUN_BYTES_OUT":
			stats.TunnelIPv6 = formatBytes(value) + " (TUN out)"
		}
	}

	return stats
}

// formatBytes converts bytes to human readable format
func formatBytes(bytesStr string) string {
	var bytes int64
	fmt.Sscanf(bytesStr, "%d", &bytes)

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Connect starts a new VPN session with the given config file
func (c *Client) Connect(configPath string) error {
	cmd := exec.Command("openvpn3", "session-start", "--config", configPath)
	return cmd.Run()
}

// Disconnect terminates a VPN session
func (c *Client) Disconnect(sessionPath string) error {
	cmd := exec.Command("openvpn3", "session-manage", "--path", sessionPath, "--disconnect")
	return cmd.Run()
}

// Pause pauses a VPN session
func (c *Client) Pause(sessionPath string) error {
	cmd := exec.Command("openvpn3", "session-manage", "--path", sessionPath, "--pause")
	return cmd.Run()
}

// Resume resumes a paused VPN session
func (c *Client) Resume(sessionPath string) error {
	cmd := exec.Command("openvpn3", "session-manage", "--path", sessionPath, "--resume")
	return cmd.Run()
}
