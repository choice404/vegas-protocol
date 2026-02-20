package internal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF4444")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF4444")).
			Padding(2, 4)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			MarginTop(1)
)

// HomeModel is the home/landing screen.
type HomeModel struct{}

func NewHomeModel() HomeModel {
	return HomeModel{}
}

func (h HomeModel) Init() tea.Cmd {
	return nil
}

func (h HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	// Handle home-screen-specific key presses and messages here.
	// Switch on msg type, update state, return commands as needed.
	return h, nil
}

func (h HomeModel) View() string {
	title := titleStyle.Render("Rebel Hacks")
	subtitle := subtitleStyle.Render("UNLV Hackathon — Theme TBD")
	hint := hintStyle.Render("Press q to quit")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		hint,
	)

	return boxStyle.Render(content)
}
