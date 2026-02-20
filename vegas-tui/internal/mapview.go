package internal

import (
	"strings"

	"rebel-hacks-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

type MapModel struct {
	scrollX  int
	scrollY  int
	dragging bool
	lastX    int
	lastY    int
	moved    bool
}

func NewMapModel() MapModel {
	return MapModel{}
}

func (m MapModel) Init() tea.Cmd {
	return nil
}

func (m MapModel) Update(msg tea.Msg) (MapModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "down", "j":
			m.scrollY++
		case "left", "h":
			if m.scrollX > 0 {
				m.scrollX--
			}
		case "right", "l":
			m.scrollX++
		}
	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				m.dragging = true
				m.lastX = msg.X
				m.lastY = msg.Y
				m.moved = false
			}
		case tea.MouseActionMotion:
			if m.dragging {
				deltaX := m.lastX - msg.X
				deltaY := m.lastY - msg.Y
				m.scrollX += deltaX
				m.scrollY += deltaY
				if m.scrollX < 0 {
					m.scrollX = 0
				}
				if m.scrollY < 0 {
					m.scrollY = 0
				}
				m.lastX = msg.X
				m.lastY = msg.Y
				if deltaX != 0 || deltaY != 0 {
					m.moved = true
				}
			}
		case tea.MouseActionRelease:
			m.dragging = false
		}

		// Wheel scroll
		if msg.Button == tea.MouseButtonWheelUp && m.scrollY > 0 {
			m.scrollY -= 3
			if m.scrollY < 0 {
				m.scrollY = 0
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			m.scrollY += 3
		}
		if msg.Button == tea.MouseButtonWheelLeft && m.scrollX > 0 {
			m.scrollX -= 3
			if m.scrollX < 0 {
				m.scrollX = 0
			}
		}
		if msg.Button == tea.MouseButtonWheelRight {
			m.scrollX += 3
		}
	}
	return m, nil
}

const mapArt = `
                    ╔══════════════════════════════════╗
                    ║     M O J A V E   W A S T E S    ║
                    ╚══════════════════════════════════╝

                               N
                               │
                          W ───┼─── E
                               │
                               S

         ┌──────────┐                      ┌──────────────┐
         │ GOODSPRNG│                      │  PRIMM       │
         │ Cemetery │                      │  Casino      │
         └────┬─────┘                      └──────┬───────┘
              │                                   │
              │          ┌───────────────┐        │
    ══════════╪══════════╡ MOJAVE OUTPOST╞════════╪═══════════
              │          └───────────────┘        │
              │                                   │
         ┌────┴─────┐   ┌───────────────┐   ┌────┴────────┐
         │ JEAN SKY │   │   NIPTON      │   │  SEARCHLGHT │
         │ DIVING   │   │ (destroyed)   │   │  AIRPORT    │
         └──────────┘   └───────┬───────┘   └─────────────┘
                                │
                    ┌───────────┴───────────┐
                    │   MOJAVE WASTELAND    │
                    │                       │
                    │    ★ YOU ARE HERE      │
                    │    (UNLV CAMPUS)       │
                    │                       │
                    └───────────┬───────────┘
                                │
              ┌─────────────────┴─────────────────┐
              │                                   │
         ┌────┴─────┐                      ┌──────┴───────┐
         │  VEGAS   │                      │   HOOVER     │
         │  STRIP   │                      │   DAM        │
         │ ♠ ♥ ♦ ♣  │                      │   ~~≈≈≈~~    │
         └──────────┘                      └──────────────┘

    ┌────────────────────────────────────────────────────────┐
    │ REBEL HACKS COMMAND CENTER                             │
    │                                                        │
    │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐      │
    │  │SCIENCE │  │STUDENT │  │ LIED   │  │ SEB    │      │
    │  │BUILDING│  │ UNION  │  │LIBRARY │  │BUILDING│      │
    │  └────────┘  └────────┘  └────────┘  └────────┘      │
    │                                                        │
    │           ╔════════════════════╗                        │
    │           ║   HACKATHON HQ    ║                        │
    │           ║   ★ BASE CAMP ★   ║                        │
    │           ╚════════════════════╝                        │
    └────────────────────────────────────────────────────────┘`

func (m MapModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" LOCAL MAP "))
	if m.dragging {
		b.WriteString(theme.AmberStyle.Render(" [DRAGGING]"))
	}
	b.WriteString("\n")

	lines := strings.Split(mapArt, "\n")

	startY := m.scrollY
	if startY >= len(lines) {
		startY = len(lines) - 1
	}

	visibleHeight := height - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	endY := startY + visibleHeight
	if endY > len(lines) {
		endY = len(lines)
	}

	for _, line := range lines[startY:endY] {
		if m.scrollX < len(line) {
			visible := line[m.scrollX:]
			if len(visible) > width-4 {
				visible = visible[:width-4]
			}
			b.WriteString(theme.BaseStyle.Render(visible))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render(" [Arrow keys/hjkl] Scroll  [Mouse drag] Pan  [Wheel] Scroll"))

	return b.String()
}
