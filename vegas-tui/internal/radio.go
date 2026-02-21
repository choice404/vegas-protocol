package internal

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type RadioStation struct {
	Name      string
	Genre     string
	Frequency string
}

type radioTickMsg struct{}

type RadioModel struct {
	stations  []RadioStation
	current   int
	playing   bool
	eqBars    []int
}

var defaultStations = []RadioStation{
	{Name: "MOJAVE MUSIC RADIO", Genre: "Classic Country & Western", Frequency: "98.5 FM"},
	{Name: "BLACK MOUNTAIN RADIO", Genre: "Dark Ambient", Frequency: "101.3 FM"},
	{Name: "NEW VEGAS RADIO", Genre: "Big Band & Swing", Frequency: "105.7 FM"},
	{Name: "RADIO NEW VEGAS", Genre: "Mr. New Vegas Hits", Frequency: "107.9 FM"},
	{Name: "REBEL HACKS FM", Genre: "Lo-Fi Beats to Code To", Frequency: "92.1 FM"},
}

func NewRadioModel() RadioModel {
	return RadioModel{
		stations: defaultStations,
		eqBars:   make([]int, 16),
	}
}

func (m RadioModel) Init() tea.Cmd {
	return nil
}

func radioTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return radioTickMsg{}
	})
}

func (m RadioModel) Update(msg tea.Msg) (RadioModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.current > 0 {
				m.current--
			}
		case "down", "j":
			if m.current < len(m.stations)-1 {
				m.current++
			}
		case "enter", " ":
			m.playing = !m.playing
			if m.playing {
				return m, radioTickCmd()
			}
		case "n":
			m.current = (m.current + 1) % len(m.stations)
		case "p":
			m.current = (m.current - 1 + len(m.stations)) % len(m.stations)
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if zone.Get("radio-play").InBounds(msg) {
				m.playing = !m.playing
				if m.playing {
					return m, radioTickCmd()
				}
			}
			if zone.Get("radio-next").InBounds(msg) {
				m.current = (m.current + 1) % len(m.stations)
			}
			if zone.Get("radio-prev").InBounds(msg) {
				m.current = (m.current - 1 + len(m.stations)) % len(m.stations)
			}
			for i := range m.stations {
				if zone.Get(fmt.Sprintf("station-%d", i)).InBounds(msg) {
					m.current = i
				}
			}
		}
	case radioTickMsg:
		if m.playing {
			// Animate equalizer bars
			for i := range m.eqBars {
				m.eqBars[i] = rand.Intn(8) + 1
			}
			return m, radioTickCmd()
		}
	}
	return m, nil
}

func (m RadioModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" RADIO "))
	b.WriteString("\n\n")

	// Current station display
	station := m.stations[m.current]
	b.WriteString(theme.DimStyle.Render("  FREQUENCY: "))
	b.WriteString(theme.AmberStyle.Render(station.Frequency))
	b.WriteString("\n\n")

	// Frequency dial visualization
	dialWidth := width - 8
	if dialWidth < 20 {
		dialWidth = 20
	}
	if dialWidth > 50 {
		dialWidth = 50
	}
	pos := float64(m.current) / float64(len(m.stations)-1) * float64(dialWidth-1)
	dial := strings.Repeat("─", int(pos)) + "▼" + strings.Repeat("─", dialWidth-1-int(pos))
	b.WriteString(fmt.Sprintf("  %s\n", theme.DimStyle.Render(dial)))
	b.WriteString("\n")

	// Station name
	b.WriteString(fmt.Sprintf("  %s\n", theme.BoldStyle.Render(station.Name)))
	b.WriteString(fmt.Sprintf("  %s\n", theme.DimStyle.Render(station.Genre)))
	b.WriteString("\n")

	// Equalizer
	if m.playing {
		var eqLine strings.Builder
		eqLine.WriteString("  ")
		for _, h := range m.eqBars {
			bar := strings.Repeat("█", h)
			pad := strings.Repeat(" ", 8-h)
			eqLine.WriteString(theme.BaseStyle.Render(bar + pad + " "))
		}
		// Show the EQ vertically (simplified: just show bar heights)
		b.WriteString(theme.BaseStyle.Render("  "))
		for _, h := range m.eqBars {
			col := ""
			for row := 8; row > 0; row-- {
				if row <= h {
					col += "█"
				} else {
					col += " "
				}
			}
			_ = col
		}
		// Simpler horizontal EQ
		b.WriteString("  ")
		for _, h := range m.eqBars {
			b.WriteString(theme.BaseStyle.Render(strings.Repeat("▮", h)))
			b.WriteString(theme.DimStyle.Render(strings.Repeat("▯", 8-h)))
			b.WriteString(" ")
		}
		b.WriteString("\n")
	} else {
		b.WriteString(theme.DimStyle.Render("  ▯▯▯▯▯▯▯▯ ▯▯▯▯▯▯▯▯ ▯▯▯▯▯▯▯▯ ▯▯▯▯▯▯▯▯"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Controls
	playBtn := "[ PLAY ]"
	if m.playing {
		playBtn = "[ PAUSE ]"
	}
	controls := fmt.Sprintf("  %s  %s  %s",
		zone.Mark("radio-prev", theme.BaseStyle.Render("[ PREV ]")),
		zone.Mark("radio-play", theme.AmberStyle.Render(playBtn)),
		zone.Mark("radio-next", theme.BaseStyle.Render("[ NEXT ]")),
	)
	b.WriteString(controls)
	b.WriteString("\n\n")

	// Station list
	b.WriteString(theme.DimStyle.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render("  STATIONS:"))
	b.WriteString("\n\n")

	for i, s := range m.stations {
		cursor := "  "
		if i == m.current {
			cursor = theme.AmberStyle.Render("> ")
		}
		nameStyle := theme.BaseStyle
		if i == m.current && m.playing {
			nameStyle = theme.AmberStyle
		}
		line := fmt.Sprintf("  %s%s  %s",
			cursor,
			nameStyle.Render(s.Name),
			theme.DimStyle.Render(s.Frequency),
		)
		line = zone.Mark(fmt.Sprintf("station-%d", i), line)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render("  [j/k] Select  [Enter/Space] Play/Pause  [n/p] Next/Prev"))

	return b.String()
}
