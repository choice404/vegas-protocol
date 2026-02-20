package internal

import (
	"rebel-hacks-tui/internal/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents the current active screen in the application.
type Screen int

const (
	ScreenHome Screen = iota
	// Add more screens here as needed:
	// ScreenDetail
	// ScreenSettings
)

// App is the root model that manages screen routing.
type App struct {
	screen Screen
	width  int
	height int
	client *client.Client

	// Screen models — add your screen models here
	home HomeModel
}

// NewApp creates the initial application state.
func NewApp() App {
	return App{
		screen: ScreenHome,
		home:   NewHomeModel(),
		client: client.New("http://localhost:8080"),
	}
}

func (a App) Init() tea.Cmd {
	return a.home.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		}
	}

	// Route updates to the active screen
	var cmd tea.Cmd
	switch a.screen {
	case ScreenHome:
		a.home, cmd = a.home.Update(msg)
	}

	return a, cmd
}

func (a App) View() string {
	var content string

	switch a.screen {
	case ScreenHome:
		content = a.home.View()
	}

	// Center content in the terminal
	return lipgloss.Place(a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
