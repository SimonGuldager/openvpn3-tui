# OpenVPN3 TUI

A terminal user interface for managing OpenVPN3 connections on Linux.

![OpenVPN3 TUI](https://img.shields.io/badge/TUI-Bubbletea-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-Linux-lightgrey)

## Features

- **Profile Management** - Save and organize your `.ovpn` configuration files with friendly names
- **Session Control** - Connect, disconnect, and monitor active VPN sessions
- **Live Statistics** - View real-time connection stats (bytes in/out, packets, tunnel IP)
- **Path Autocomplete** - Tab-completion when adding new profiles
- **Duplicate Prevention** - Prevents connecting to the same VPN twice
- **Theme Support** - Integrates with [Omarchy](https://omarchy.org/) themes with hot-reload
- **Waybar Integration** - Optional status indicator for your bar

## Screenshots

```
OpenVPN3 TUI
Profiles  Sessions

> Work VPN [connected]
  Home Server
  Client VPN

tab: switch view • j/k: navigate • enter: connect • a: add • d: delete • r: refresh • q: quit
```

## Installation

### Prerequisites

- [OpenVPN3 Linux](https://openvpn.net/cloud-docs/openvpn-3-client-for-linux/) installed and configured
- Go 1.21+ (for building from source)

### Build from source

```bash
git clone https://github.com/SimonGuldager/openvpn3-tui.git
cd openvpn3-tui
go build -o openvpn3-tui .
```

### Install

```bash
# User-local installation (recommended)
cp openvpn3-tui ~/.local/bin/

# Or system-wide
sudo cp openvpn3-tui /usr/local/bin/
```

## Usage

```bash
openvpn3-tui
```

### Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch between Profiles and Sessions |
| `j` / `k` or `↑` / `↓` | Navigate list |
| `Enter` | Connect (profiles) / Show stats (sessions) |
| `a` | Add new profile |
| `d` | Delete profile / Disconnect session |
| `s` | Show session statistics |
| `r` | Refresh sessions |
| `q` | Quit |

### Adding Profiles

1. Press `a` to add a new profile
2. Type the path to your `.ovpn` file (supports `~` and tab-completion)
3. Press `Enter` and provide a friendly name
4. The profile is saved and ready to use

Profiles are stored in `~/.config/openvpn3-tui/config.json`.

## Theme Support

OpenVPN3 TUI supports theming via a simple TOML configuration file.

### Theme File Location

```
~/.config/openvpn3-tui/theme.toml
```

### Theme Format

```toml
accent = "#89b4fa"
foreground = "#cdd6f4"
background = "#1e1e2e"
selection_foreground = "#1e1e2e"
selection_background = "#89b4fa"

success = "#a6e3a1"
warning = "#f9e2af"
error = "#f38ba8"
muted = "#585b70"
```

### Omarchy Integration

For [Omarchy](https://omarchy.org/) users, theme integration is automatic:

1. Create the template file `~/.config/omarchy/themed/openvpn3-tui.toml.tpl`:

```toml
accent = "{{ accent }}"
foreground = "{{ foreground }}"
background = "{{ background }}"
selection_foreground = "{{ selection_foreground }}"
selection_background = "{{ accent }}"

success = "{{ color2 }}"
warning = "{{ color3 }}"
error = "{{ color1 }}"
muted = "{{ color8 }}"
```

2. Create a symlink:

```bash
mkdir -p ~/.config/openvpn3-tui
ln -sf ~/.config/omarchy/current/theme/openvpn3-tui.toml ~/.config/openvpn3-tui/theme.toml
```

3. Apply your current theme to generate the file:

```bash
omarchy-theme-set "$(omarchy-theme-current)"
```

The TUI will hot-reload colors when the theme changes.

## Waybar Integration

Add a VPN status indicator to your Waybar:

### Status Script

Create `~/.local/bin/openvpn3-waybar-status`:

```bash
#!/bin/bash
sessions=$(openvpn3 sessions-list 2>/dev/null)
count=$(echo "$sessions" | grep -c "Path:" || true)

if [[ $count -gt 0 ]]; then
    configs=$(echo "$sessions" | grep "Config name:" | sed 's/.*Config name:[[:space:]]*//' | \
        sed 's/(Config not available)//' | while read -r line; do
        basename "$line" .ovpn 2>/dev/null
    done | paste -sd ',' | sed 's/,/, /g')

    tooltip="VPN Connected: $configs"
    echo "{\"text\": \"󰌆\", \"tooltip\": \"$tooltip\", \"class\": \"connected\"}"
else
    echo "{\"text\": \"󰌊\", \"tooltip\": \"VPN Disconnected\", \"class\": \"disconnected\"}"
fi
```

```bash
chmod +x ~/.local/bin/openvpn3-waybar-status
```

### Waybar Config

Add to `~/.config/waybar/config.jsonc`:

```jsonc
"custom/openvpn": {
    "exec": "/path/to/openvpn3-waybar-status",
    "return-type": "json",
    "format": "{}",
    "interval": 5,
    "on-click": "openvpn3-tui",  // or your preferred launcher
    "tooltip": true
}
```

Add to `~/.config/waybar/style.css`:

```css
#custom-openvpn {
    min-width: 12px;
    margin: 0 7.5px;
}

#custom-openvpn.connected {
    color: #a6e3a1;
}

#custom-openvpn.disconnected {
    opacity: 0.5;
}
```

## Project Structure

```
openvpn3-tui/
├── main.go                 # Entry point
├── go.mod / go.sum         # Dependencies
└── internal/
    ├── config/
    │   └── config.go       # Profile persistence
    ├── openvpn/
    │   └── client.go       # OpenVPN3 CLI wrapper
    └── ui/
        ├── model.go        # TUI model and logic
        ├── styles.go       # Lipgloss styling
        ├── theme.go        # Theme loading and hot-reload
        └── completer.go    # Path autocomplete
```

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [fsnotify](https://github.com/fsnotify/fsnotify) - File watching for theme hot-reload

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

- Built for use with [OpenVPN3 Linux](https://github.com/OpenVPN/openvpn3-linux)
- Theme system designed for [Omarchy](https://omarchy.org/) integration
- TUI powered by the excellent [Charm](https://charm.sh/) libraries
