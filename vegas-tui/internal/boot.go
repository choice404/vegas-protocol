package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type bootTickMsg struct{}

var bootSteps = []struct {
	label  string
	result string
}{
	{"BIOS CHECK", "OK"},
	{"MEMORY ALLOCATION", "OK"},
	{"SENSOR ARRAY CALIBRATION", "OK"},
	{"NETWORK INTERFACE", "OK"},
	{"AI CORE LINK", "STANDBY"},
	{"PERSONALITY MATRIX", "LOADED"},
	{"HOLOTAPE DRIVER", "OK"},
	{"RADIATION SENSORS", "NOMINAL"},
}

type BootModel struct {
	step     int
	done     bool
	width    int
	height   int
}

func NewBootModel() BootModel {
	return BootModel{}
}

func (m BootModel) Init() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return bootTickMsg{}
	})
}

func (m BootModel) Update(msg tea.Msg) (BootModel, tea.Cmd) {
	switch msg.(type) {
	case bootTickMsg:
		m.step++
		if m.step > len(bootSteps)+2 {
			m.done = true
			return m, nil
		}
		return m, tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
			return bootTickMsg{}
		})
	}
	return m, nil
}

func (m BootModel) View(width, height int) string {
	var b strings.Builder

	// Logo
	logo := theme.BoldStyle.Render(theme.Logo)
	b.WriteString(logo)
	b.WriteString("\n")

	// Subtitle
	subtitle := theme.AmberStyle.Render("  " + theme.Subtitle)
	b.WriteString(subtitle)
	b.WriteString("\n")

	// Divider
	b.WriteString(theme.DimStyle.Render("  " + theme.Divider))
	b.WriteString("\n\n")

	// Boot steps
	for i, step := range bootSteps {
		if i >= m.step {
			break
		}
		dots := strings.Repeat(".", 32-len(step.label))
		resultStyle := theme.BaseStyle
		if step.result == "STANDBY" {
			resultStyle = theme.AmberStyle
		}
		line := fmt.Sprintf("  > %s %s %s",
			theme.BaseStyle.Render(step.label),
			theme.DimStyle.Render(dots),
			resultStyle.Render(step.result),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Progress bar
	if m.step > 0 {
		b.WriteString("\n")
		progress := m.step
		if progress > len(bootSteps) {
			progress = len(bootSteps)
		}
		pct := float64(progress) / float64(len(bootSteps))
		barWidth := 40
		filled := int(pct * float64(barWidth))
		bar := theme.GaugeFilled.Render(strings.Repeat("█", filled)) +
			theme.GaugeEmpty.Render(strings.Repeat("░", barWidth-filled))
		b.WriteString(fmt.Sprintf("  [%s] %s",
			bar,
			theme.BaseStyle.Render(fmt.Sprintf("%d%%", int(pct*100))),
		))
		b.WriteString("\n")
	}

	// Ready message
	if m.step > len(bootSteps) {
		b.WriteString("\n")
		b.WriteString(theme.AmberStyle.Render("  SYSTEM READY. WELCOME, COURIER."))
		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render("  Press any key to continue..."))
	}

	content := b.String()
	box := theme.BorderStyle.
		Width(56).
		Render(content)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}
