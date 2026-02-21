package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type questFocus int

const (
	focusQuestList questFocus = iota
	focusTaskList
	focusAddQuest
	focusAddTask
)

// questSavedMsg signals quests were saved to disk.
type questSavedMsg struct{}

// QuestFromAIMsg is sent when the LLM creates a quest via JSON.
type QuestFromAIMsg struct {
	Quest settings.QuestLine
}

type QuestsModel struct {
	quests      []settings.QuestLine
	questCursor int
	taskCursor  int
	focus       questFocus
	input       textinput.Model
	scrollY     int
}

func NewQuestsModel() QuestsModel {
	ti := textinput.New()
	ti.Placeholder = "Enter name..."
	ti.CharLimit = 100
	ti.Width = 40

	return QuestsModel{
		quests: settings.LoadQuests(),
		input:  ti,
	}
}

func (m QuestsModel) Init() tea.Cmd {
	return nil
}

func (m QuestsModel) Update(msg tea.Msg) (QuestsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case QuestFromAIMsg:
		m.quests = append(m.quests, msg.Quest)
		return m, m.saveQuests()

	case questSavedMsg:
		// Saved successfully, nothing to do
		return m, nil

	case tea.KeyMsg:
		// If we're in an input mode, handle text input
		if m.focus == focusAddQuest || m.focus == focusAddTask {
			switch msg.String() {
			case "enter":
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					if m.focus == focusAddQuest {
						m.quests = append(m.quests, settings.QuestLine{
							ID:          settings.GenerateQuestID(val),
							Name:        strings.ToUpper(val),
							Description: "",
							Priority:    "medium",
							CreatedAt:   time.Now().Format(time.RFC3339),
							Tasks:       []settings.QuestTask{},
						})
						m.questCursor = len(m.quests) - 1
					} else if m.focus == focusAddTask && m.questCursor < len(m.quests) {
						m.quests[m.questCursor].Tasks = append(m.quests[m.questCursor].Tasks, settings.QuestTask{
							Name:     val,
							Done:     false,
							Priority: "medium",
						})
					}
				}
				m.input.Reset()
				m.focus = focusTaskList
				m.input.Blur()
				return m, m.saveQuests()
			case "esc":
				m.input.Reset()
				m.focus = focusTaskList
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "up", "k":
			if m.focus == focusQuestList {
				if m.questCursor > 0 {
					m.questCursor--
					m.taskCursor = 0
				}
			} else {
				if m.taskCursor > 0 {
					m.taskCursor--
				}
			}
		case "down", "j":
			if m.focus == focusQuestList {
				if m.questCursor < len(m.quests)-1 {
					m.questCursor++
					m.taskCursor = 0
				}
			} else if m.questCursor < len(m.quests) {
				q := m.quests[m.questCursor]
				if m.taskCursor < len(q.Tasks)-1 {
					m.taskCursor++
				}
			}
		case "enter", " ":
			if m.focus == focusQuestList {
				m.focus = focusTaskList
				m.taskCursor = 0
			} else if m.questCursor < len(m.quests) {
				q := &m.quests[m.questCursor]
				if m.taskCursor < len(q.Tasks) {
					q.Tasks[m.taskCursor].Done = !q.Tasks[m.taskCursor].Done
					return m, m.saveQuests()
				}
			}
		case "esc", "backspace":
			if m.focus == focusTaskList {
				m.focus = focusQuestList
			}
		case "a":
			// Add task to current questline
			if m.focus == focusTaskList && m.questCursor < len(m.quests) {
				m.focus = focusAddTask
				m.input.Placeholder = "New task name..."
				m.input.Focus()
				return m, textinput.Blink
			}
		case "A":
			// Add new questline
			m.focus = focusAddQuest
			m.input.Placeholder = "New questline name..."
			m.input.Focus()
			return m, textinput.Blink
		case "d":
			// Delete task or questline
			if m.focus == focusTaskList && m.questCursor < len(m.quests) {
				q := &m.quests[m.questCursor]
				if m.taskCursor < len(q.Tasks) {
					q.Tasks = append(q.Tasks[:m.taskCursor], q.Tasks[m.taskCursor+1:]...)
					if m.taskCursor >= len(q.Tasks) && m.taskCursor > 0 {
						m.taskCursor--
					}
					return m, m.saveQuests()
				}
			} else if m.focus == focusQuestList && m.questCursor < len(m.quests) {
				m.quests = append(m.quests[:m.questCursor], m.quests[m.questCursor+1:]...)
				if m.questCursor >= len(m.quests) && m.questCursor > 0 {
					m.questCursor--
				}
				return m, m.saveQuests()
			}
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			// Button clicks
			if zone.Get("quest-new").InBounds(msg) {
				m.focus = focusAddQuest
				m.input.Placeholder = "New questline name..."
				m.input.Focus()
				return m, textinput.Blink
			}
			if zone.Get("quest-add-task").InBounds(msg) && m.focus == focusTaskList && m.questCursor < len(m.quests) {
				m.focus = focusAddTask
				m.input.Placeholder = "New task name..."
				m.input.Focus()
				return m, textinput.Blink
			}
			if zone.Get("quest-delete").InBounds(msg) {
				if m.focus == focusTaskList && m.questCursor < len(m.quests) {
					q := &m.quests[m.questCursor]
					if m.taskCursor < len(q.Tasks) {
						q.Tasks = append(q.Tasks[:m.taskCursor], q.Tasks[m.taskCursor+1:]...)
						if m.taskCursor >= len(q.Tasks) && m.taskCursor > 0 {
							m.taskCursor--
						}
						return m, m.saveQuests()
					}
				} else if m.focus == focusQuestList && m.questCursor < len(m.quests) {
					m.quests = append(m.quests[:m.questCursor], m.quests[m.questCursor+1:]...)
					if m.questCursor >= len(m.quests) && m.questCursor > 0 {
						m.questCursor--
					}
					return m, m.saveQuests()
				}
			}
			if zone.Get("quest-back").InBounds(msg) && m.focus == focusTaskList {
				m.focus = focusQuestList
				return m, nil
			}
			for i := range m.quests {
				if zone.Get(fmt.Sprintf("ql-%d", i)).InBounds(msg) {
					m.questCursor = i
					m.focus = focusTaskList
					m.taskCursor = 0
					return m, nil
				}
			}
			if m.questCursor < len(m.quests) {
				for i := range m.quests[m.questCursor].Tasks {
					if zone.Get(fmt.Sprintf("qt-%d", i)).InBounds(msg) {
						m.taskCursor = i
						m.focus = focusTaskList
						m.quests[m.questCursor].Tasks[i].Done = !m.quests[m.questCursor].Tasks[i].Done
						return m, m.saveQuests()
					}
				}
			}
		}
		// Wheel scroll
		if msg.Button == tea.MouseButtonWheelUp {
			if m.focus == focusQuestList && m.questCursor > 0 {
				m.questCursor--
			} else if m.taskCursor > 0 {
				m.taskCursor--
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.focus == focusQuestList && m.questCursor < len(m.quests)-1 {
				m.questCursor++
			} else if m.questCursor < len(m.quests) && m.taskCursor < len(m.quests[m.questCursor].Tasks)-1 {
				m.taskCursor++
			}
		}
	}
	return m, nil
}

func (m QuestsModel) saveQuests() tea.Cmd {
	quests := m.quests
	return func() tea.Msg {
		_ = settings.SaveQuests(quests)
		return questSavedMsg{}
	}
}

func (m QuestsModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" QUEST LOG "))
	b.WriteString("\n\n")

	if len(m.quests) == 0 {
		b.WriteString(theme.DimStyle.Render("  No questlines. Press [A] to create one."))
		b.WriteString("\n")
		return b.String()
	}

	// Questline list
	b.WriteString(theme.DimStyle.Render("  QUESTLINES:"))
	b.WriteString("\n")

	for i, q := range m.quests {
		cursor := "  "
		if i == m.questCursor {
			if m.focus == focusQuestList {
				cursor = theme.AmberStyle.Render("> ")
			} else {
				cursor = theme.BaseStyle.Render("> ")
			}
		}

		done := 0
		for _, t := range q.Tasks {
			if t.Done {
				done++
			}
		}
		total := len(q.Tasks)
		pct := 0.0
		if total > 0 {
			pct = float64(done) / float64(total)
		}

		// Mini progress bar
		barW := 8
		filled := int(pct * float64(barW))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)

		nameStyle := theme.BaseStyle
		if i == m.questCursor {
			nameStyle = theme.AmberStyle
		}

		priorityIcon := ""
		switch q.Priority {
		case "high":
			priorityIcon = theme.RedStyle.Render("!")
		case "medium":
			priorityIcon = theme.AmberStyle.Render("-")
		default:
			priorityIcon = theme.DimStyle.Render(" ")
		}

		line := fmt.Sprintf("  %s%s %s [%d/%d] %s",
			cursor,
			priorityIcon,
			nameStyle.Render(q.Name),
			done, total,
			theme.DimStyle.Render(bar),
		)
		line = zone.Mark(fmt.Sprintf("ql-%d", i), line)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Task list for selected questline
	if m.questCursor < len(m.quests) {
		q := m.quests[m.questCursor]
		b.WriteString("\n")

		divW := width - 6
		if divW < 20 {
			divW = 20
		}
		if divW > 50 {
			divW = 50
		}
		b.WriteString(theme.DimStyle.Render("  " + strings.Repeat("─", divW)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s", theme.BoldStyle.Render(q.Name)))
		if q.Description != "" {
			b.WriteString(fmt.Sprintf(" — %s", theme.DimStyle.Render(q.Description)))
		}
		b.WriteString("\n\n")

		if len(q.Tasks) == 0 {
			b.WriteString(theme.DimStyle.Render("  No tasks yet. Press [a] to add one."))
			b.WriteString("\n")
		}

		for i, t := range q.Tasks {
			cursor := "  "
			if i == m.taskCursor && m.focus == focusTaskList {
				cursor = theme.AmberStyle.Render("> ")
			}

			checkbox := theme.DimStyle.Render("[ ]")
			nameStyle := theme.BaseStyle
			if t.Done {
				checkbox = theme.BaseStyle.Render("[x]")
				nameStyle = theme.DimStyle
			}

			priorityTag := ""
			switch t.Priority {
			case "high":
				priorityTag = theme.RedStyle.Render(" [!]")
			case "low":
				priorityTag = theme.DimStyle.Render(" [~]")
			}

			line := fmt.Sprintf("  %s%s %s%s", cursor, checkbox, nameStyle.Render(t.Name), priorityTag)
			line = zone.Mark(fmt.Sprintf("qt-%d", i), line)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Input area
	if m.focus == focusAddQuest || m.focus == focusAddTask {
		b.WriteString("\n")
		label := "  NEW QUESTLINE: "
		if m.focus == focusAddTask {
			label = "  NEW TASK: "
		}
		b.WriteString(theme.AmberStyle.Render(label))
		b.WriteString(m.input.View())
		b.WriteString("\n")
	}

	// Buttons
	b.WriteString("\n  ")
	b.WriteString(zone.Mark("quest-new", theme.AmberStyle.Render("[ NEW QUEST ]")))
	if m.focus == focusTaskList && m.questCursor < len(m.quests) {
		b.WriteString("  ")
		b.WriteString(zone.Mark("quest-add-task", theme.BaseStyle.Render("[ ADD TASK ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("quest-delete", theme.RedStyle.Render("[ DELETE ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("quest-back", theme.DimStyle.Render("[ BACK ]")))
	} else if m.focus == focusQuestList && len(m.quests) > 0 {
		b.WriteString("  ")
		b.WriteString(zone.Mark("quest-delete", theme.RedStyle.Render("[ DELETE ]")))
	}

	// Help
	b.WriteString("\n\n")
	if m.focus == focusQuestList {
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] View Tasks  [A] New Quest  [d] Delete"))
	} else if m.focus == focusTaskList {
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter/Space] Toggle  [a] Add Task  [d] Delete  [Esc] Back"))
	} else {
		b.WriteString(theme.DimStyle.Render("  [Enter] Confirm  [Esc] Cancel"))
	}

	return b.String()
}
