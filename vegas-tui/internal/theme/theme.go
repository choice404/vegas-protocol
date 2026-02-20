package theme

import "github.com/charmbracelet/lipgloss"

// Pip-Boy color palette
var (
	Green     = lipgloss.Color("#00FF00")
	DarkGreen = lipgloss.Color("#008F00")
	DimGreen  = lipgloss.Color("#005500")
	Amber     = lipgloss.Color("#FFBF00")
	Red       = lipgloss.Color("#FF4444")
	Black     = lipgloss.Color("#000000")
	White     = lipgloss.Color("#FFFFFF")
)

// Base text styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(Green)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimGreen)

	BoldStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Underline(true)

	AmberStyle = lipgloss.NewStyle().
			Foreground(Amber)

	RedStyle = lipgloss.NewStyle().
			Foreground(Red)
)

// Tab styles
var (
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(Black).
			Background(Green).
			Bold(true).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(Green).
				Padding(0, 2)

	TabBarStyle = lipgloss.NewStyle().
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Green)
)

// Container styles
var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Green)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(DimGreen).
			Padding(0, 1)

	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

// Gauge styles
var (
	GaugeFilled = lipgloss.NewStyle().
			Foreground(Green)

	GaugeEmpty = lipgloss.NewStyle().
			Foreground(DimGreen)

	GaugeLabel = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Width(14)

	GaugeWarning = lipgloss.NewStyle().
			Foreground(Amber)

	GaugeCritical = lipgloss.NewStyle().
			Foreground(Red)
)

// Boot sequence logo
const Logo = `
 ██╗   ██╗ ███████╗ ██████╗  █████╗ ███████╗
 ██║   ██║ ██╔════╝██╔════╝ ██╔══██╗██╔════╝
 ██║   ██║ █████╗  ██║  ███╗███████║███████╗
 ╚██╗ ██╔╝ ██╔══╝  ██║   ██║██╔══██║╚════██║
  ╚████╔╝  ███████╗╚██████╔╝██║  ██║███████║
   ╚═══╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝`

const Subtitle = "VIRTUAL ELECTRONIC GENERAL ASSISTANT SYSTEM"

const Divider = "═══════════════════════════════════════════════"
