package internal

import (
	"fmt"
	"strings"

	"rebel-hacks-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type Item struct {
	Name        string
	Category    string
	Description string
	Weight      string
	Value       string
}

type ItemsModel struct {
	items    []Item
	cursor   int
	selected int
}

var inventory = []Item{
	{
		Name:        "Pip-Boy 5000",
		Category:    "APPAREL",
		Description: "The device you're using right now. A marvel of pre-war engineering. Touch-enabled and radiation-hardened.",
		Weight:      "3 lbs",
		Value:       "---",
	},
	{
		Name:        "Stimpak",
		Category:    "AID",
		Description: "A quick dose of restorative compounds. Heals minor wounds and restores focus during long coding sessions.",
		Weight:      "1 lb",
		Value:       "75 caps",
	},
	{
		Name:        "Nuka-Cola Quantum",
		Category:    "AID",
		Description: "The radioactive glow means it's working. +20 AP, +200 Rads. Tastes like victory and isotopes.",
		Weight:      "1 lb",
		Value:       "100 caps",
	},
	{
		Name:        "10mm Pistol",
		Category:    "WEAPONS",
		Description: "Standard issue sidearm. Reliable, accurate, and just intimidating enough for the Mojave.",
		Weight:      "3 lbs",
		Value:       "200 caps",
	},
	{
		Name:        "Bobby Pin (x14)",
		Category:    "MISC",
		Description: "Essential for lockpicking. You never know when you'll find a locked terminal or footlocker.",
		Weight:      "0 lbs",
		Value:       "1 cap",
	},
	{
		Name:        "Rad-X",
		Category:    "AID",
		Description: "Provides temporary radiation resistance. Take before entering server rooms with poor ventilation.",
		Weight:      "0 lbs",
		Value:       "40 caps",
	},
	{
		Name:        "Sunset Sarsaparilla",
		Category:    "AID",
		Description: "A refreshing wasteland beverage. Collect the star bottle caps for a special prize!",
		Weight:      "1 lb",
		Value:       "5 caps",
	},
	{
		Name:        "Holotape: VEGAS.exe",
		Category:    "MISC",
		Description: "Contains the V.E.G.A.S. Protocol source code. Handle with care - this is the only copy.",
		Weight:      "0 lbs",
		Value:       "---",
	},
	{
		Name:        "NCR Radio Encryption Key",
		Category:    "MISC",
		Description: "Allows decryption of NCR military broadcasts. May contain useful intelligence.",
		Weight:      "0 lbs",
		Value:       "250 caps",
	},
}

func NewItemsModel() ItemsModel {
	return ItemsModel{
		items:    inventory,
		selected: -1,
	}
}

func (m ItemsModel) Init() tea.Cmd {
	return nil
}

func (m ItemsModel) Update(msg tea.Msg) (ItemsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if m.selected == m.cursor {
				m.selected = -1
			} else {
				m.selected = m.cursor
			}
		}
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			for i := range m.items {
				if zone.Get(fmt.Sprintf("item-%d", i)).InBounds(msg) {
					m.cursor = i
					if m.selected == i {
						m.selected = -1
					} else {
						m.selected = i
					}
				}
			}
		}
		// Wheel scroll
		if msg.Button == tea.MouseButtonWheelUp && m.cursor > 0 {
			m.cursor--
		}
		if msg.Button == tea.MouseButtonWheelDown && m.cursor < len(m.items)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m ItemsModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" INVENTORY "))
	b.WriteString("\n\n")

	// Item list
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = theme.AmberStyle.Render("> ")
		}

		nameStyle := theme.BaseStyle
		if i == m.selected {
			nameStyle = theme.AmberStyle
		}

		cat := theme.DimStyle.Render(fmt.Sprintf("[%s]", item.Category))
		line := fmt.Sprintf(" %s%s %s", cursor, nameStyle.Render(item.Name), cat)
		line = zone.Mark(fmt.Sprintf("item-%d", i), line)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Detail panel
	if m.selected >= 0 && m.selected < len(m.items) {
		item := m.items[m.selected]
		detailWidth := width - 6
		if detailWidth < 30 {
			detailWidth = 30
		}
		if detailWidth > 60 {
			detailWidth = 60
		}

		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render(" " + strings.Repeat("─", detailWidth)))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf(" %s\n", theme.BoldStyle.Render(item.Name)))
		b.WriteString(fmt.Sprintf(" %s\n\n", theme.DimStyle.Render(item.Category)))

		// Word-wrap description
		desc := wrapText(item.Description, detailWidth-2)
		for _, line := range strings.Split(desc, "\n") {
			b.WriteString(fmt.Sprintf(" %s\n", theme.BaseStyle.Render(line)))
		}

		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(" %s %s    %s %s\n",
			theme.DimStyle.Render("WGT:"),
			theme.BaseStyle.Render(item.Weight),
			theme.DimStyle.Render("VAL:"),
			theme.AmberStyle.Render(item.Value),
		))
	}

	b.WriteString("\n")
	b.WriteString(theme.DimStyle.Render(" [j/k] Navigate  [Enter/Click] Inspect"))

	return b.String()
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}
