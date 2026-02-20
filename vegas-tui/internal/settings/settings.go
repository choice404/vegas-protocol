package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const configDirName = "vegas-protocol"

// Settings holds user preferences persisted to disk.
type Settings struct {
	Editor      string   `json:"editor"`
	ServerURL   string   `json:"server_url"`
	OllamaURL   string   `json:"ollama_url"`
	OllamaModel string   `json:"ollama_model"`
	Theme       string   `json:"theme"`
	ProjectDirs []string `json:"project_dirs"`
}

// QuestLine is a project/questline containing multiple tasks.
type QuestLine struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Priority    string      `json:"priority"`
	Tasks       []QuestTask `json:"tasks"`
	CreatedAt   string      `json:"created_at"`
}

// QuestTask is a single task within a questline.
type QuestTask struct {
	Name     string `json:"name"`
	Done     bool   `json:"done"`
	Priority string `json:"priority"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", configDirName)
}

func detectEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "nano"
}

// DefaultSettings returns sensible defaults.
func DefaultSettings() *Settings {
	return &Settings{
		Editor:      detectEditor(),
		ServerURL:   "http://localhost:8080",
		OllamaURL:   "http://localhost:11434",
		OllamaModel: "llama3",
		Theme:       "green",
		ProjectDirs: []string{},
	}
}

// Load reads settings from disk, returning defaults if file doesn't exist.
func Load() *Settings {
	path := filepath.Join(configPath(), "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultSettings()
	}
	s := DefaultSettings()
	_ = json.Unmarshal(data, s)
	// Ensure editor is set
	if s.Editor == "" {
		s.Editor = detectEditor()
	}
	return s
}

// Save writes settings to disk.
func Save(s *Settings) error {
	dir := configPath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling settings: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "settings.json"), data, 0644)
}

// LoadQuests reads questlines from disk.
func LoadQuests() []QuestLine {
	path := filepath.Join(configPath(), "quests.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultQuests()
	}
	var quests []QuestLine
	if err := json.Unmarshal(data, &quests); err != nil {
		return DefaultQuests()
	}
	return quests
}

// SaveQuests writes questlines to disk.
func SaveQuests(quests []QuestLine) error {
	dir := configPath()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(quests, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling quests: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "quests.json"), data, 0644)
}

// DefaultQuests returns initial questlines for new users.
func DefaultQuests() []QuestLine {
	return []QuestLine{
		{
			ID:          "rebel-hacks-2025",
			Name:        "REBEL HACKS 2025",
			Description: "UNLV Hackathon - V.E.G.A.S. Protocol",
			Priority:    "high",
			CreatedAt:   time.Now().Format(time.RFC3339),
			Tasks: []QuestTask{
				{Name: "Register at Check-in", Done: false, Priority: "high"},
				{Name: "Set Up Dev Environment", Done: false, Priority: "high"},
				{Name: "Build Core TUI Features", Done: false, Priority: "high"},
				{Name: "Connect to AI Mainframe", Done: false, Priority: "medium"},
				{Name: "Implement Project Management", Done: false, Priority: "medium"},
				{Name: "Polish the Interface", Done: false, Priority: "medium"},
				{Name: "Prepare Demo", Done: false, Priority: "high"},
				{Name: "Submit Project", Done: false, Priority: "high"},
			},
		},
	}
}

// GenerateQuestID makes a simple ID from a name + timestamp.
func GenerateQuestID(name string) string {
	ts := time.Now().Unix()
	safe := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			safe += string(c)
		} else if c >= 'A' && c <= 'Z' {
			safe += string(c - 'A' + 'a')
		} else if c == ' ' {
			safe += "-"
		}
	}
	if len(safe) > 30 {
		safe = safe[:30]
	}
	return fmt.Sprintf("%s-%d", safe, ts%10000)
}
