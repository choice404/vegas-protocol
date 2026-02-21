package internal

import (
	"fmt"
	"strings"
	"time"

	"rebel-hacks-tui/internal/settings"
	"rebel-hacks-tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type AppState int

const (
	StateBoot AppState = iota
	StateMain
)

const (
	TabStats    = 0
	TabItems    = 1
	TabData     = 2
	TabQuests   = 3
	TabProjects = 4
	TabRadio    = 5
	TabMap      = 6
	TabSettings = 7
	TabExit= 8
)

var tabNames = []string{"STATS", "ITEMS", "DATA", "QUESTS", "PROJ", "RADIO", "MAP", "SET", "EXIT"}

type App struct {
	state     AppState
	activeTab int
	width     int
	height    int

	// Sub-models
	boot     BootModel
	stats    StatsModel
	items    ItemsModel
	data     DataModel
	quests   QuestsModel
	projects ProjectsModel
	radio    RadioModel
	mapV     MapModel
	settings SettingsModel

	// Shared settings
	appSettings *settings.Settings
}

func NewApp() App {
	s := settings.Load()

	return App{
		state:       StateBoot,
		appSettings: s,
		boot:        NewBootModel(),
		stats:       NewStatsModel(),
		items:       NewItemsModel(),
		data:        NewDataModel(s.ServerURL, s.OllamaModel),
		quests:      NewQuestsModel(),
		projects:    NewProjectsModel(s),
		radio:       NewRadioModel(),
		mapV:        NewMapModel(),
		settings:    NewSettingsModel(s),
	}
}

func (a App) Init() tea.Cmd {
	return a.boot.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	// Quest from AI gets routed to the quests model
	case QuestFromAIMsg:
		var cmd tea.Cmd
		a.quests, cmd = a.quests.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Boot state: any key to continue after boot completes
		if a.state == StateBoot {
			if a.boot.done {
				a.state = StateMain
				return a, tea.Batch(
					a.stats.Init(),
					a.data.Init(),
					a.projects.Init(),
				)
			}
			var cmd tea.Cmd
			a.boot, cmd = a.boot.Update(msg)
			return a, cmd
		}

		// Don't intercept keys when an input is focused
		if a.isInputFocused() {
			cmd := a.updateActiveTab(msg)
			return a, cmd
		}

		// Main state key handling
		switch msg.String() {
		case "q":
			return a, tea.Quit
		case "tab":
			a.switchTab((a.activeTab + 1) % len(tabNames))
			return a, nil
		case "shift+tab":
			a.switchTab((a.activeTab - 1 + len(tabNames)) % len(tabNames))
			return a, nil
		case "1":
			a.switchTab(TabStats)
			return a, nil
		case "2":
			a.switchTab(TabItems)
			return a, nil
		case "3":
			a.switchTab(TabData)
			return a, nil
		case "4":
			a.switchTab(TabQuests)
			return a, nil
		case "5":
			a.switchTab(TabProjects)
			return a, nil
		case "6":
			a.switchTab(TabRadio)
			return a, nil
		case "7":
			a.switchTab(TabMap)
			return a, nil
		case "8":
			a.switchTab(TabSettings)
			return a, nil
		case "9":
			return a, tea.Quit
		}

	case tea.MouseMsg:
		if a.state == StateMain && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			for i := range tabNames {
				if zone.Get(fmt.Sprintf("tab-%d", i)).InBounds(msg) {
					a.switchTab(i)
					return a, nil
				}
			}
		}
	}

	// Route updates to active model
	var cmd tea.Cmd
	switch a.state {
	case StateBoot:
		a.boot, cmd = a.boot.Update(msg)
	case StateMain:
		cmd = a.updateActiveTab(msg)
	}

	return a, cmd
}

func (a *App) isInputFocused() bool {
	switch a.activeTab {
	case TabData:
		return a.data.Focused()
	case TabQuests:
		return a.quests.focus == focusAddQuest || a.quests.focus == focusAddTask
	case TabProjects:
		return a.projects.InputFocused()
	case TabSettings:
		return a.settings.InputFocused()
	}
	return false
}

func (a *App) switchTab(tab int) {
	// Blur inputs when leaving tabs
	if a.activeTab == TabData {
		a.data.Blur()
	}
	a.activeTab = tab
	// Focus chat input when entering DATA tab
	if tab == TabData {
		a.data.Focus()
	}
}

func (a *App) updateActiveTab(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch a.activeTab {
	case TabStats:
		a.stats, cmd = a.stats.Update(msg)
	case TabItems:
		a.items, cmd = a.items.Update(msg)
	case TabData:
		a.data, cmd = a.data.Update(msg)
	case TabQuests:
		a.quests, cmd = a.quests.Update(msg)
	case TabProjects:
		a.projects, cmd = a.projects.Update(msg)
	case TabRadio:
		a.radio, cmd = a.radio.Update(msg)
	case TabMap:
		a.mapV, cmd = a.mapV.Update(msg)
	case TabSettings:
		a.settings, cmd = a.settings.Update(msg)
	case TabExit:
		return tea.Quit
	}
	return cmd
}

func (a App) View() string {
	if a.state == StateBoot {
		return zone.Scan(a.boot.View(a.width, a.height))
	}

	// Header
	header := a.renderHeader()

	// Tab bar
	tabBar := a.renderTabBar()

	// Content area
	contentHeight := a.height - 6
	if contentHeight < 5 {
		contentHeight = 5
	}
	contentWidth := a.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	content := a.renderContent(contentWidth, contentHeight)

	// Footer
	footer := a.renderFooter()

	full := lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		theme.ContentStyle.
			Width(a.width-2).
			Height(contentHeight).
			Render(content),
		footer,
	)

	return zone.Scan(full)
}

func (a App) renderHeader() string {
	title := theme.BoldStyle.Render(" V.E.G.A.S. PROTOCOL v1.0 ")
	padding := a.width - lipgloss.Width(title) - 2
	if padding < 0 {
		padding = 0
	}
	return theme.HeaderStyle.
		Width(a.width).
		Render(title + strings.Repeat(" ", padding))
}

func (a App) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		id := fmt.Sprintf("tab-%d", i)
		var tab string
		if i == a.activeTab {
			tab = zone.Mark(id, theme.ActiveTabStyle.Render(name))
		} else {
			tab = zone.Mark(id, theme.InactiveTabStyle.Render(name))
		}
		tabs = append(tabs, tab)
	}

	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return theme.TabBarStyle.Width(a.width).Render(tabRow)
}

func (a App) renderContent(width, height int) string {
	switch a.activeTab {
	case TabStats:
		return a.stats.View(width, height)
	case TabItems:
		return a.items.View(width, height)
	case TabData:
		return a.data.View(width, height)
	case TabQuests:
		return a.quests.View(width, height)
	case TabProjects:
		return a.projects.View(width, height)
	case TabRadio:
		return a.radio.View(width, height)
	case TabMap:
		return a.mapV.View(width, height)
	case TabSettings:
		return a.settings.View(width, height)
	}
	return ""
}

func (a App) renderFooter() string {
	now := time.Now().Format("15:04:05")
	status := theme.BaseStyle.Render("⚡ ONLINE")
	clock := theme.DimStyle.Render(now)
	tabHint := theme.DimStyle.Render("[Tab] Switch  [1-8] Jump  [q] Quit")

	left := fmt.Sprintf(" %s │ %s", status, clock)
	right := tabHint + " "

	padding := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 0 {
		padding = 0
	}

	return theme.FooterStyle.
		Width(a.width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.DimGreen).
		Render(left + strings.Repeat(" ", padding) + right)
}
