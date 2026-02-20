package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"rebel-hacks-tui/internal/settings"
	"rebel-hacks-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type projectFocus int

const (
	focusSetupDir projectFocus = iota
	focusProjectList
	focusFileList
	focusSearching
)

type editorDoneMsg struct{ err error }

type ProjectEntry struct {
	Name string
	Path string
}

type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
}

type ProjectsModel struct {
	appSettings   *settings.Settings
	focus         projectFocus
	projects      []ProjectEntry
	files         []FileEntry
	projectCursor int
	fileCursor    int
	search        textinput.Model
	dirInput      textinput.Model
	searchFilter  string
	currentDir    string // Currently browsed project directory
}

func NewProjectsModel(s *settings.Settings) ProjectsModel {
	searchTI := textinput.New()
	searchTI.Placeholder = "Search projects..."
	searchTI.CharLimit = 100
	searchTI.Width = 40

	dirTI := textinput.New()
	dirTI.Placeholder = "/path/to/your/projects"
	dirTI.CharLimit = 200
	dirTI.Width = 50

	m := ProjectsModel{
		appSettings: s,
		search:      searchTI,
		dirInput:    dirTI,
	}

	if len(s.ProjectDirs) > 0 {
		m.focus = focusProjectList
		m.loadProjects()
	} else {
		m.focus = focusSetupDir
		m.dirInput.Focus()
	}

	return m
}

func (m *ProjectsModel) loadProjects() {
	m.projects = nil
	for _, dir := range m.appSettings.ProjectDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				m.projects = append(m.projects, ProjectEntry{
					Name: e.Name(),
					Path: filepath.Join(dir, e.Name()),
				})
			}
		}
	}
	sort.Slice(m.projects, func(i, j int) bool {
		return m.projects[i].Name < m.projects[j].Name
	})
}

func (m *ProjectsModel) loadFiles(projectPath string) {
	m.files = nil
	m.currentDir = projectPath
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return
	}
	// Directories first, then files
	var dirs, files []FileEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fe := FileEntry{
			Name:  e.Name(),
			Path:  filepath.Join(projectPath, e.Name()),
			IsDir: e.IsDir(),
		}
		if e.IsDir() {
			dirs = append(dirs, fe)
		} else {
			files = append(files, fe)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	m.files = append(dirs, files...)
}

func (m ProjectsModel) filteredProjects() []ProjectEntry {
	if m.searchFilter == "" {
		return m.projects
	}
	lower := strings.ToLower(m.searchFilter)
	var filtered []ProjectEntry
	for _, p := range m.projects {
		if strings.Contains(strings.ToLower(p.Name), lower) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (m ProjectsModel) Init() tea.Cmd {
	if m.focus == focusSetupDir {
		return textinput.Blink
	}
	return nil
}

func (m ProjectsModel) Update(msg tea.Msg) (ProjectsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case editorDoneMsg:
		// Editor closed, return to file view
		return m, nil

	case tea.KeyMsg:
		// Setup directory input
		if m.focus == focusSetupDir {
			switch msg.String() {
			case "enter":
				dir := strings.TrimSpace(m.dirInput.Value())
				if dir != "" {
					// Expand ~ to home dir
					if strings.HasPrefix(dir, "~") {
						home, _ := os.UserHomeDir()
						dir = filepath.Join(home, dir[1:])
					}
					if info, err := os.Stat(dir); err == nil && info.IsDir() {
						m.appSettings.ProjectDirs = append(m.appSettings.ProjectDirs, dir)
						_ = settings.Save(m.appSettings)
						m.loadProjects()
						m.focus = focusProjectList
						m.dirInput.Blur()
						m.dirInput.Reset()
					}
				}
				return m, nil
			case "esc":
				if len(m.appSettings.ProjectDirs) > 0 {
					m.focus = focusProjectList
					m.dirInput.Blur()
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.dirInput, cmd = m.dirInput.Update(msg)
			return m, cmd
		}

		// Search mode
		if m.focus == focusSearching {
			switch msg.String() {
			case "enter", "esc":
				m.searchFilter = m.search.Value()
				m.focus = focusProjectList
				m.search.Blur()
				m.projectCursor = 0
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.searchFilter = m.search.Value()
			m.projectCursor = 0
			return m, cmd
		}

		// Project or file list navigation
		switch msg.String() {
		case "/":
			if m.focus == focusProjectList {
				m.focus = focusSearching
				m.search.Focus()
				return m, textinput.Blink
			}
		case "up", "k":
			if m.focus == focusProjectList {
				if m.projectCursor > 0 {
					m.projectCursor--
				}
			} else if m.focus == focusFileList {
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			}
		case "down", "j":
			if m.focus == focusProjectList {
				filtered := m.filteredProjects()
				if m.projectCursor < len(filtered)-1 {
					m.projectCursor++
				}
			} else if m.focus == focusFileList {
				if m.fileCursor < len(m.files)-1 {
					m.fileCursor++
				}
			}
		case "enter":
			if m.focus == focusProjectList {
				filtered := m.filteredProjects()
				if m.projectCursor < len(filtered) {
					m.loadFiles(filtered[m.projectCursor].Path)
					m.focus = focusFileList
					m.fileCursor = 0
				}
			} else if m.focus == focusFileList && m.fileCursor < len(m.files) {
				f := m.files[m.fileCursor]
				if f.IsDir {
					m.loadFiles(f.Path)
					m.fileCursor = 0
				}
			}
		case "e":
			// Open file in editor
			if m.focus == focusFileList && m.fileCursor < len(m.files) {
				f := m.files[m.fileCursor]
				if !f.IsDir {
					editor := m.appSettings.Editor
					c := exec.Command(editor, f.Path)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return editorDoneMsg{err}
					})
				}
			}
		case "esc", "backspace":
			if m.focus == focusFileList {
				// Go up a directory or back to project list
				parent := filepath.Dir(m.currentDir)
				isProjectRoot := false
				for _, p := range m.projects {
					if p.Path == m.currentDir {
						isProjectRoot = true
						break
					}
				}
				if isProjectRoot {
					m.focus = focusProjectList
					m.files = nil
				} else {
					m.loadFiles(parent)
					m.fileCursor = 0
				}
			} else if m.focus == focusProjectList {
				m.searchFilter = ""
				m.search.Reset()
			}
		case "P":
			// Add another project directory
			m.focus = focusSetupDir
			m.dirInput.Focus()
			return m, textinput.Blink
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if m.focus == focusProjectList {
				filtered := m.filteredProjects()
				for i := range filtered {
					if zone.Get(fmt.Sprintf("proj-%d", i)).InBounds(msg) {
						m.projectCursor = i
						m.loadFiles(filtered[i].Path)
						m.focus = focusFileList
						m.fileCursor = 0
						return m, nil
					}
				}
			} else if m.focus == focusFileList {
				for i := range m.files {
					if zone.Get(fmt.Sprintf("file-%d", i)).InBounds(msg) {
						m.fileCursor = i
						f := m.files[i]
						if f.IsDir {
							m.loadFiles(f.Path)
							m.fileCursor = 0
						}
						return m, nil
					}
				}
			}
		}
		// Wheel scroll
		if msg.Button == tea.MouseButtonWheelUp {
			if m.focus == focusProjectList && m.projectCursor > 0 {
				m.projectCursor--
			} else if m.focus == focusFileList && m.fileCursor > 0 {
				m.fileCursor--
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.focus == focusProjectList {
				filtered := m.filteredProjects()
				if m.projectCursor < len(filtered)-1 {
					m.projectCursor++
				}
			} else if m.focus == focusFileList && m.fileCursor < len(m.files)-1 {
				m.fileCursor++
			}
		}
	}
	return m, nil
}

func (m ProjectsModel) View(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" PROJECTS "))
	b.WriteString("\n\n")

	if m.focus == focusSetupDir {
		b.WriteString(theme.AmberStyle.Render("  CONFIGURE PROJECT DIRECTORY"))
		b.WriteString("\n\n")
		b.WriteString(theme.BaseStyle.Render("  Enter a directory that contains your projects:"))
		b.WriteString("\n\n")
		b.WriteString("  " + theme.AmberStyle.Render("> ") + m.dirInput.View())
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  Each subdirectory will appear as a separate project."))
		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render("  Example: /home/user/projects"))
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  [Enter] Confirm  [Esc] Cancel"))
		return b.String()
	}

	if m.focus == focusSearching || m.searchFilter != "" {
		b.WriteString("  " + theme.DimStyle.Render("SEARCH: "))
		if m.focus == focusSearching {
			b.WriteString(m.search.View())
		} else {
			b.WriteString(theme.AmberStyle.Render(m.searchFilter))
		}
		b.WriteString("\n\n")
	}

	// Show project directories
	if len(m.appSettings.ProjectDirs) > 0 {
		for _, d := range m.appSettings.ProjectDirs {
			b.WriteString(fmt.Sprintf("  %s %s\n", theme.DimStyle.Render("ROOT:"), theme.BaseStyle.Render(d)))
		}
		b.WriteString("\n")
	}

	if m.focus == focusProjectList || m.focus == focusSearching {
		filtered := m.filteredProjects()
		if len(filtered) == 0 {
			b.WriteString(theme.DimStyle.Render("  No projects found."))
			b.WriteString("\n")
		}
		for i, p := range filtered {
			cursor := "  "
			if i == m.projectCursor {
				cursor = theme.AmberStyle.Render("> ")
			}
			nameStyle := theme.BaseStyle
			if i == m.projectCursor {
				nameStyle = theme.AmberStyle
			}
			line := fmt.Sprintf("  %s%s  %s",
				cursor,
				nameStyle.Render(p.Name),
				theme.DimStyle.Render(p.Path),
			)
			line = zone.Mark(fmt.Sprintf("proj-%d", i), line)
			b.WriteString(line)
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Open  [/] Search  [P] Add Dir"))
	}

	if m.focus == focusFileList {
		// Show breadcrumb
		b.WriteString(fmt.Sprintf("  %s %s\n\n",
			theme.DimStyle.Render("PATH:"),
			theme.BaseStyle.Render(m.currentDir),
		))

		if len(m.files) == 0 {
			b.WriteString(theme.DimStyle.Render("  Empty directory."))
			b.WriteString("\n")
		}

		maxShow := height - 12
		if maxShow < 5 {
			maxShow = 5
		}
		start := 0
		if m.fileCursor >= maxShow {
			start = m.fileCursor - maxShow + 1
		}
		end := start + maxShow
		if end > len(m.files) {
			end = len(m.files)
		}

		for i := start; i < end; i++ {
			f := m.files[i]
			cursor := "  "
			if i == m.fileCursor {
				cursor = theme.AmberStyle.Render("> ")
			}

			icon := "  "
			nameStyle := theme.BaseStyle
			if f.IsDir {
				icon = theme.AmberStyle.Render("/ ")
				if i == m.fileCursor {
					nameStyle = theme.AmberStyle
				}
			} else if i == m.fileCursor {
				nameStyle = theme.AmberStyle
			}

			line := fmt.Sprintf("  %s%s%s", cursor, icon, nameStyle.Render(f.Name))
			line = zone.Mark(fmt.Sprintf("file-%d", i), line)
			b.WriteString(line)
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Open Dir  [e] Edit File  [Esc] Back"))
	}

	return b.String()
}

func (m ProjectsModel) InputFocused() bool {
	return m.focus == focusSetupDir || m.focus == focusSearching
}
