package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type Message struct {
	Sender  string
	Content string
}

type chatResponseMsg struct {
	content string
	err     error
}

type DataModel struct {
	messages    []Message
	textInput   textinput.Model
	viewport    viewport.Model
	loading     bool
	serverURL   string
	ollamaModel string
	width       int
	height      int
}

func NewDataModel(serverURL, ollamaModel string) DataModel {
	ti := textinput.New()
	ti.Placeholder = "Type your query, Courier..."
	ti.CharLimit = 500
	ti.Width = 50

	return DataModel{
		textInput:   ti,
		serverURL:   serverURL,
		ollamaModel: ollamaModel,
		messages: []Message{
			{
				Sender:  "V.E.G.A.S.",
				Content: "Systems online. How can I assist you today, Courier? I can also create quest plans — just ask me to plan a project.",
			},
		},
	}
}

func (m DataModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m DataModel) Update(msg tea.Msg) (DataModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if zone.Get("data-send").InBounds(msg) {
				if !m.loading && strings.TrimSpace(m.textInput.Value()) != "" {
					prompt := m.textInput.Value()
					m.messages = append(m.messages, Message{Sender: "USER", Content: prompt})
					m.textInput.Reset()
					m.loading = true
					m.updateViewport()
					cmds = append(cmds, m.sendChat(prompt))
					return m, tea.Batch(cmds...)
				}
			}
			if zone.Get("data-clear").InBounds(msg) {
				m.messages = []Message{{
					Sender:  "V.E.G.A.S.",
					Content: "Terminal cleared. How can I assist you, Courier?",
				}}
				m.updateViewport()
				return m, nil
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.loading || strings.TrimSpace(m.textInput.Value()) == "" {
				break
			}
			prompt := m.textInput.Value()
			m.messages = append(m.messages, Message{Sender: "USER", Content: prompt})
			m.textInput.Reset()
			m.loading = true
			m.updateViewport()
			cmds = append(cmds, m.sendChat(prompt))
		}
	case chatResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.messages = append(m.messages, Message{
				Sender:  "V.E.G.A.S.",
				Content: "CONNECTION LOST - RAD INTERFERENCE. " + msg.err.Error(),
			})
		} else {
			// Check for quest JSON in the response
			cleaned, quest := parseQuestFromResponse(msg.content)
			m.messages = append(m.messages, Message{
				Sender:  "V.E.G.A.S.",
				Content: cleaned,
			})
			if quest != nil {
				// Send quest to the app for adding to quest system
				cmds = append(cmds, func() tea.Msg {
					return QuestFromAIMsg{Quest: *quest}
				})
			}
		}
		m.updateViewport()
	}

	// Update text input
	var tiCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

var questJSONRegex = regexp.MustCompile("(?s)```vegas-quest\\s*\\n(.*?)```")

// parseQuestFromResponse extracts a quest JSON block from the AI response.
// Returns the cleaned text (without the JSON block) and the parsed quest.
func parseQuestFromResponse(content string) (string, *settings.QuestLine) {
	matches := questJSONRegex.FindStringSubmatch(content)
	if matches == nil {
		return content, nil
	}

	jsonStr := strings.TrimSpace(matches[1])
	var quest settings.QuestLine
	if err := json.Unmarshal([]byte(jsonStr), &quest); err != nil {
		return content, nil
	}

	// Generate ID if missing
	if quest.ID == "" {
		quest.ID = settings.GenerateQuestID(quest.Name)
	}
	if quest.CreatedAt == "" {
		quest.CreatedAt = time.Now().Format(time.RFC3339)
	}
	if quest.Name != "" {
		quest.Name = strings.ToUpper(quest.Name)
	}

	// Remove the JSON block from displayed text
	cleaned := questJSONRegex.ReplaceAllString(content, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		cleaned = fmt.Sprintf("Quest '%s' has been added to your Quest Log, Courier.", quest.Name)
	}

	return cleaned, &quest
}

func (m *DataModel) updateViewport() {
	content := m.renderMessages()
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m DataModel) renderMessages() string {
	var b strings.Builder
	wrapWidth := m.width - 10
	if wrapWidth < 20 {
		wrapWidth = 20
	}
	
	for _, msg := range m.messages {
		if msg.Sender == "USER" {
			b.WriteString(theme.AmberStyle.Render(fmt.Sprintf(" > %s: ", msg.Sender)))
			wrapped := wrapText(msg.Content, wrapWidth)
			lines := strings.Split(wrapped, "\n")
			for i, line := range lines {
				if i == 0 {
					b.WriteString(theme.BaseStyle.Render(line))
				} else {
					b.WriteString("\n   ")
					b.WriteString(theme.BaseStyle.Render(line))
				}
			}
		} else {
			b.WriteString(theme.BoldStyle.Render(fmt.Sprintf(" > %s: ", msg.Sender)))
			wrapped := wrapText(msg.Content, wrapWidth)
			lines := strings.Split(wrapped, "\n")
			for i, line := range lines {
				if i == 0 {
					b.WriteString(theme.BaseStyle.Render(line))
				} else {
					b.WriteString("\n")
					b.WriteString(theme.BaseStyle.Render("   " + line))
				}
			}
		}
		b.WriteString("\n\n")
	}
	if m.loading {
		b.WriteString(theme.DimStyle.Render(" > V.E.G.A.S.: Processing query..."))
		b.WriteString("\n")
	}
	return b.String()
}

func (m *DataModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" V.E.G.A.S. TERMINAL "))
	b.WriteString("\n\n")

	vpHeight := height - 8
	if vpHeight < 5 {
		vpHeight = 5
	}
	if width != m.width || height != m.height {
		m.viewport = viewport.New(width-4, vpHeight)
		m.width = width
		m.height = height
		m.updateViewport()
	}

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	divWidth := width - 4
	if divWidth < 10 {
		divWidth = 10
	}
	b.WriteString(theme.DimStyle.Render(strings.Repeat("─", divWidth)))
	b.WriteString("\n")

	b.WriteString(m.textInput.View())
	b.WriteString("  ")
	b.WriteString(zone.Mark("data-send", theme.AmberStyle.Render("[ SEND ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("data-clear", theme.BaseStyle.Render("[ CLEAR ]")))

	return b.String()
}

func (m DataModel) Focused() bool {
	return m.textInput.Focused()
}

func (m *DataModel) Focus() tea.Cmd {
	return m.textInput.Focus()
}

func (m *DataModel) Blur() {
	m.textInput.Blur()
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

const systemPrompt = `You are V.E.G.A.S. (Virtual Electronic General Assistant System), an AI assistant in the style of a Fallout: New Vegas computer terminal. You speak with a retro-futuristic, slightly formal tone. You call the user "Courier" and occasionally reference the Mojave Wasteland. Keep responses concise and helpful.

QUEST CREATION: When the user asks you to plan a project, create a task list, or organize work, include a quest definition in your response using this exact format:

` + "```vegas-quest" + `
{
  "name": "Quest Name Here",
  "description": "Brief description",
  "priority": "high",
  "tasks": [
    {"name": "First task", "done": false, "priority": "high"},
    {"name": "Second task", "done": false, "priority": "medium"}
  ]
}
` + "```" + `

This will automatically add the quest to the Courier's Quest Log. Only include this block when the user explicitly asks for project planning or task creation.`

func (m DataModel) sendChat(prompt string) tea.Cmd {
	serverURL := m.serverURL
	model := m.ollamaModel
	return func() tea.Msg {
		reqBody := ollamaRequest{
			Model:  model,
			Prompt: prompt,
			System: systemPrompt,
			Stream: false,
		}

		body, _ := json.Marshal(reqBody)
		client := &http.Client{Timeout: 120 * time.Second}

		resp, err := client.Post(serverURL+"/api/chat", "application/json", bytes.NewReader(body))
		if err != nil {
			return chatResponseMsg{err: fmt.Errorf("server unreachable")}
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return chatResponseMsg{err: fmt.Errorf("failed to read response")}
		}

		if resp.StatusCode != http.StatusOK {
			// Try to parse JSON error from server
			var errResp struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
				return chatResponseMsg{err: fmt.Errorf("%s", errResp.Error)}
			}
			return chatResponseMsg{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))}
		}

		var ollamaResp ollamaResponse
		if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
			return chatResponseMsg{err: fmt.Errorf("failed to parse response: %s", string(respBody))}
		}

		return chatResponseMsg{content: ollamaResp.Response}
	}
}
