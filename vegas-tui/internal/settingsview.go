package internal

import (
	"fmt"
	"strings"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type settingsSavedMsg struct{}

type settingEntry struct {
	Key    string
	Label  string
	Value  string
	IsBool bool
}

type SettingsModel struct {
	appSettings *settings.Settings
	entries     []settingEntry
	cursor      int
	editing     bool
	input       textinput.Model
	dirty       bool
}

func NewSettingsModel(s *settings.Settings) SettingsModel {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 50

	m := SettingsModel{
		appSettings: s,
		input:       ti,
	}
	m.refreshEntries()
	return m
}

func boolToStr(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func (m *SettingsModel) refreshEntries() {
	s := m.appSettings
	m.entries = []settingEntry{
		{Key: "editor", Label: "EDITOR", Value: s.Editor},
		{Key: "server_url", Label: "SERVER URL", Value: s.ServerURL},
		{Key: "ollama_url", Label: "OLLAMA URL", Value: s.OllamaURL},
		{Key: "ollama_model", Label: "OLLAMA MODEL", Value: s.OllamaModel},
		{Key: "theme", Label: "THEME", Value: s.Theme},
		{Key: "check_updates", Label: "CHECK UPDATES", Value: boolToStr(s.CheckUpdates), IsBool: true},
		{Key: "auto_update", Label: "AUTO UPDATE", Value: boolToStr(s.AutoUpdate), IsBool: true},
	}
	// Add project dirs as separate entries
	for i, d := range s.ProjectDirs {
		m.entries = append(m.entries, settingEntry{
			Key:   fmt.Sprintf("project_dir_%d", i),
			Label: fmt.Sprintf("PROJECT DIR %d", i+1),
			Value: d,
		})
	}
}

func (m *SettingsModel) applyEntry(idx int, val string) {
	if idx >= len(m.entries) {
		return
	}
	key := m.entries[idx].Key
	s := m.appSettings

	switch key {
	case "editor":
		s.Editor = val
	case "server_url":
		s.ServerURL = val
	case "ollama_url":
		s.OllamaURL = val
	case "ollama_model":
		s.OllamaModel = val
	case "theme":
		if val == "green" || val == "amber" {
			s.Theme = val
		}
	case "check_updates":
		s.CheckUpdates = !s.CheckUpdates
	case "auto_update":
		s.AutoUpdate = !s.AutoUpdate
	default:
		if strings.HasPrefix(key, "project_dir_") {
			// Parse index from key
			var i int
			fmt.Sscanf(key, "project_dir_%d", &i)
			if i < len(s.ProjectDirs) {
				s.ProjectDirs[i] = val
			}
		}
	}
	m.refreshEntries()
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsSavedMsg:
		m.dirty = false
		return m, nil

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				val := m.input.Value()
				m.applyEntry(m.cursor, val)
				m.editing = false
				m.dirty = true
				m.input.Blur()
				return m, nil
			case "esc":
				m.editing = false
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.entries) {
				if m.entries[m.cursor].IsBool {
					m.applyEntry(m.cursor, "")
					m.dirty = true
					return m, nil
				}
				m.editing = true
				m.input.SetValue(m.entries[m.cursor].Value)
				m.input.Focus()
				return m, textinput.Blink
			}
		case "s":
			if m.dirty {
				s := m.appSettings
				return m, func() tea.Msg {
					_ = settings.Save(s)
					return settingsSavedMsg{}
				}
			}
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if zone.Get("set-save").InBounds(msg) && m.dirty {
				s := m.appSettings
				return m, func() tea.Msg {
					_ = settings.Save(s)
					return settingsSavedMsg{}
				}
			}
			if zone.Get("set-reset").InBounds(msg) {
				defaults := settings.DefaultSettings()
				m.appSettings.Editor = defaults.Editor
				m.appSettings.ServerURL = defaults.ServerURL
				m.appSettings.OllamaURL = defaults.OllamaURL
				m.appSettings.OllamaModel = defaults.OllamaModel
				m.appSettings.Theme = defaults.Theme
				m.appSettings.CheckUpdates = defaults.CheckUpdates
				m.appSettings.AutoUpdate = defaults.AutoUpdate
				m.dirty = true
				m.refreshEntries()
				return m, nil
			}
			for i := range m.entries {
				if zone.Get(fmt.Sprintf("set-%d", i)).InBounds(msg) {
					m.cursor = i
					return m, nil
				}
			}
		}
		if msg.Button == tea.MouseButtonWheelUp && m.cursor > 0 {
			m.cursor--
		}
		if msg.Button == tea.MouseButtonWheelDown && m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m SettingsModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" SETTINGS "))
	if m.dirty {
		b.WriteString(theme.AmberStyle.Render(" (unsaved)"))
	}
	b.WriteString("\n\n")

	labelWidth := 16
	for i, e := range m.entries {
		cursor := "  "
		if i == m.cursor {
			cursor = theme.AmberStyle.Render("> ")
		}

		label := theme.DimStyle.Render(fmt.Sprintf("%-*s", labelWidth, e.Label+":"))

		var val string
		if m.editing && i == m.cursor {
			val = m.input.View()
		} else {
			valStyle := theme.BaseStyle
			if i == m.cursor {
				valStyle = theme.AmberStyle
			}
			val = valStyle.Render(e.Value)
		}

		line := fmt.Sprintf("  %s%s %s", cursor, label, val)
		if !m.editing || i != m.cursor {
			line = zone.Mark(fmt.Sprintf("set-%d", i), line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Separators between sections
		if i < len(m.entries)-1 {
			divW := width - 8
			if divW < 20 {
				divW = 20
			}
			if divW > 50 {
				divW = 50
			}
			if e.Key == "theme" {
				b.WriteString("\n")
				b.WriteString(theme.DimStyle.Render("  " + strings.Repeat("─", divW)))
				b.WriteString("\n")
				b.WriteString(theme.DimStyle.Render("  UPDATES:"))
				b.WriteString("\n\n")
			}
			if e.Key == "auto_update" {
				b.WriteString("\n")
				b.WriteString(theme.DimStyle.Render("  " + strings.Repeat("─", divW)))
				b.WriteString("\n")
				b.WriteString(theme.DimStyle.Render("  PROJECT DIRECTORIES:"))
				b.WriteString("\n\n")
			}
		}
	}

	// Buttons
	b.WriteString("\n  ")
	if m.dirty {
		b.WriteString(zone.Mark("set-save", theme.AmberStyle.Render("[ SAVE ]")))
		b.WriteString("  ")
	}
	b.WriteString(zone.Mark("set-reset", theme.BaseStyle.Render("[ RESET DEFAULTS ]")))

	b.WriteString("\n\n")
	if m.editing {
		b.WriteString(theme.DimStyle.Render("  [Enter] Save value  [Esc] Cancel"))
	} else {
		helpParts := []string{"[j/k] Navigate", "[Enter] Edit"}
		if m.dirty {
			helpParts = append(helpParts, "[s] Save to disk")
		}
		b.WriteString(theme.DimStyle.Render("  " + strings.Join(helpParts, "  ")))
	}

	return b.String()
}

func (m SettingsModel) InputFocused() bool {
	return m.editing
}

// Settings returns a pointer to the underlying settings for other models to read.
func (m SettingsModel) Settings() *settings.Settings {
	return m.appSettings
}
