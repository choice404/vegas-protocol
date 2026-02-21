package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/settings"
	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

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
	focusGitStatus
	focusCommitMsg
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

	// Git integration
	gitStatus   GitStatus
	gitLoading  bool
	gitFeedback string
	commitInput textinput.Model
	projectRoot string // Root path of the currently open project
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

	commitTI := textinput.New()
	commitTI.Placeholder = "Enter commit message..."
	commitTI.CharLimit = 200
	commitTI.Width = 50

	m := ProjectsModel{
		appSettings: s,
		search:      searchTI,
		dirInput:    dirTI,
		commitInput: commitTI,
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

func (m *ProjectsModel) enterProject(path string) tea.Cmd {
	m.loadFiles(path)
	m.projectRoot = path
	m.focus = focusFileList
	m.fileCursor = 0
	m.gitStatus = GitStatus{}
	m.gitFeedback = ""
	return fetchGitStatus(path)
}

func (m ProjectsModel) Update(msg tea.Msg) (ProjectsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case editorDoneMsg:
		return m, nil

	// Git message handlers
	case gitStatusMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "STATUS ERROR: " + msg.err.Error()
		}
		m.gitStatus = msg.status
		return m, nil

	case gitStageMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "STAGE FAILED: " + msg.err.Error()
		} else {
			m.gitFeedback = "STAGED ALL CHANGES"
		}
		return m, fetchGitStatus(m.projectRoot)

	case gitCommitMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "COMMIT FAILED: " + msg.err.Error()
		} else {
			m.gitFeedback = "COMMITTED"
		}
		return m, fetchGitStatus(m.projectRoot)

	case gitPushMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "PUSH FAILED: " + msg.err.Error()
		} else {
			m.gitFeedback = "PUSHED"
		}
		return m, fetchGitStatus(m.projectRoot)

	case gitPullMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "PULL FAILED: " + msg.err.Error()
		} else {
			m.gitFeedback = "PULLED"
			m.loadFiles(m.currentDir)
		}
		return m, fetchGitStatus(m.projectRoot)

	case gitFetchMsg:
		m.gitLoading = false
		if msg.err != nil {
			m.gitFeedback = "FETCH FAILED: " + msg.err.Error()
		} else {
			m.gitFeedback = "FETCHED"
		}
		return m, fetchGitStatus(m.projectRoot)

	case tea.KeyMsg:
		// Setup directory input
		if m.focus == focusSetupDir {
			switch msg.String() {
			case "enter":
				dir := strings.TrimSpace(m.dirInput.Value())
				if dir != "" {
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

		// Commit message input
		if m.focus == focusCommitMsg {
			switch msg.String() {
			case "enter":
				message := strings.TrimSpace(m.commitInput.Value())
				if message != "" {
					m.gitLoading = true
					m.gitFeedback = "COMMITTING..."
					m.commitInput.Reset()
					m.commitInput.Blur()
					m.focus = focusGitStatus
					return m, gitCommit(m.projectRoot, message)
				}
				return m, nil
			case "esc":
				m.commitInput.Reset()
				m.commitInput.Blur()
				m.focus = focusGitStatus
				return m, nil
			}
			var cmd tea.Cmd
			m.commitInput, cmd = m.commitInput.Update(msg)
			return m, cmd
		}

		// Git status view
		if m.focus == focusGitStatus {
			if !m.gitLoading {
				switch msg.String() {
				case "s":
					m.gitLoading = true
					m.gitFeedback = "STAGING..."
					return m, gitStageAll(m.projectRoot)
				case "c":
					if len(m.gitStatus.Staged) > 0 {
						m.focus = focusCommitMsg
						m.commitInput.Focus()
						return m, textinput.Blink
					}
					m.gitFeedback = "NOTHING STAGED TO COMMIT"
					return m, nil
				case "p":
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "PUSHING..."
						return m, gitPush(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				case "l":
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "PULLING..."
						return m, gitPull(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				case "f":
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "FETCHING..."
						return m, gitFetch(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				case "esc":
					m.focus = focusFileList
					m.gitFeedback = ""
					return m, nil
				}
			}
			return m, nil
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
					cmd := m.enterProject(filtered[m.projectCursor].Path)
					return m, cmd
				}
			} else if m.focus == focusFileList && m.fileCursor < len(m.files) {
				f := m.files[m.fileCursor]
				if f.IsDir {
					m.loadFiles(f.Path)
					m.fileCursor = 0
				}
			}
		case "e":
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
		case "g":
			if m.focus == focusFileList && m.gitStatus.IsRepo {
				m.focus = focusGitStatus
				m.gitFeedback = ""
				return m, fetchGitStatus(m.projectRoot)
			}
		case "esc", "backspace":
			if m.focus == focusFileList {
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
					m.gitStatus = GitStatus{}
				} else {
					m.loadFiles(parent)
					m.fileCursor = 0
				}
			} else if m.focus == focusProjectList {
				m.searchFilter = ""
				m.search.Reset()
			}
		case "P":
			m.focus = focusSetupDir
			m.dirInput.Focus()
			return m, textinput.Blink
		}

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			// Button clicks
			if zone.Get("proj-add-dir").InBounds(msg) {
				m.focus = focusSetupDir
				m.dirInput.Focus()
				return m, textinput.Blink
			}
			if zone.Get("proj-search").InBounds(msg) && m.focus == focusProjectList {
				m.focus = focusSearching
				m.search.Focus()
				return m, textinput.Blink
			}
			if zone.Get("proj-back").InBounds(msg) {
				if m.focus == focusGitStatus {
					m.focus = focusFileList
					m.gitFeedback = ""
					return m, nil
				}
				if m.focus == focusFileList {
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
						m.gitStatus = GitStatus{}
					} else {
						m.loadFiles(parent)
						m.fileCursor = 0
					}
					return m, nil
				}
			}
			if zone.Get("proj-edit").InBounds(msg) && m.focus == focusFileList && m.fileCursor < len(m.files) {
				f := m.files[m.fileCursor]
				if !f.IsDir {
					editor := m.appSettings.Editor
					c := exec.Command(editor, f.Path)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return editorDoneMsg{err}
					})
				}
			}

			// Git button: open git status view
			if zone.Get("proj-git").InBounds(msg) && m.focus == focusFileList && m.gitStatus.IsRepo {
				m.focus = focusGitStatus
				m.gitFeedback = ""
				return m, fetchGitStatus(m.projectRoot)
			}

			// Git action buttons (only in git status view)
			if m.focus == focusGitStatus && !m.gitLoading {
				if zone.Get("proj-stage-all").InBounds(msg) {
					m.gitLoading = true
					m.gitFeedback = "STAGING..."
					return m, gitStageAll(m.projectRoot)
				}
				if zone.Get("proj-commit").InBounds(msg) {
					if len(m.gitStatus.Staged) > 0 {
						m.focus = focusCommitMsg
						m.commitInput.Focus()
						return m, textinput.Blink
					}
					m.gitFeedback = "NOTHING STAGED TO COMMIT"
					return m, nil
				}
				if zone.Get("proj-push").InBounds(msg) {
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "PUSHING..."
						return m, gitPush(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				}
				if zone.Get("proj-pull").InBounds(msg) {
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "PULLING..."
						return m, gitPull(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				}
				if zone.Get("proj-fetch").InBounds(msg) {
					if m.gitStatus.HasRemote {
						m.gitLoading = true
						m.gitFeedback = "FETCHING..."
						return m, gitFetch(m.projectRoot)
					}
					m.gitFeedback = "NO REMOTE CONFIGURED"
					return m, nil
				}
			}

			if m.focus == focusProjectList {
				filtered := m.filteredProjects()
				for i := range filtered {
					if zone.Get(fmt.Sprintf("proj-%d", i)).InBounds(msg) {
						m.projectCursor = i
						cmd := m.enterProject(filtered[i].Path)
						return m, cmd
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

		b.WriteString("\n  ")
		b.WriteString(zone.Mark("proj-search", theme.AmberStyle.Render("[ SEARCH ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-add-dir", theme.BaseStyle.Render("[ ADD DIR ]")))
		b.WriteString("\n\n")
		b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Open  [/] Search  [P] Add Dir"))
	}

	if m.focus == focusGitStatus || m.focus == focusCommitMsg {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			theme.DimStyle.Render("PROJECT:"),
			theme.BaseStyle.Render(m.projectRoot),
		))
		b.WriteString(fmt.Sprintf("  %s %s\n\n",
			theme.DimStyle.Render("BRANCH:"),
			theme.AmberStyle.Render(m.gitStatus.Branch),
		))

		// Staged files
		if len(m.gitStatus.Staged) > 0 {
			b.WriteString(theme.BaseStyle.Render("  STAGED:"))
			b.WriteString("\n")
			for _, f := range m.gitStatus.Staged {
				b.WriteString(fmt.Sprintf("    %s %s\n", theme.BaseStyle.Render("+"), theme.BaseStyle.Render(f)))
			}
			b.WriteString("\n")
		}

		// Unstaged (modified) files
		if len(m.gitStatus.Unstaged) > 0 {
			b.WriteString(theme.AmberStyle.Render("  MODIFIED:"))
			b.WriteString("\n")
			for _, f := range m.gitStatus.Unstaged {
				b.WriteString(fmt.Sprintf("    %s %s\n", theme.AmberStyle.Render("~"), theme.AmberStyle.Render(f)))
			}
			b.WriteString("\n")
		}

		// Untracked files
		if len(m.gitStatus.Untracked) > 0 {
			b.WriteString(theme.DimStyle.Render("  UNTRACKED:"))
			b.WriteString("\n")
			for _, f := range m.gitStatus.Untracked {
				b.WriteString(fmt.Sprintf("    %s %s\n", theme.DimStyle.Render("?"), theme.DimStyle.Render(f)))
			}
			b.WriteString("\n")
		}

		if m.gitStatus.Clean {
			b.WriteString(theme.BaseStyle.Render("  WORKING TREE CLEAN"))
			b.WriteString("\n\n")
		}

		// Commit message input
		if m.focus == focusCommitMsg {
			b.WriteString("  " + theme.AmberStyle.Render("COMMIT MSG: ") + m.commitInput.View())
			b.WriteString("\n\n")
		}

		// Feedback line
		if m.gitFeedback != "" {
			style := theme.BaseStyle
			if strings.Contains(m.gitFeedback, "FAILED") || strings.Contains(m.gitFeedback, "ERROR") || strings.Contains(m.gitFeedback, "NO ") || strings.Contains(m.gitFeedback, "NOTHING") {
				style = theme.RedStyle
			} else if strings.Contains(m.gitFeedback, "...") {
				style = theme.AmberStyle
			}
			b.WriteString("  " + style.Render(m.gitFeedback))
			b.WriteString("\n\n")
		}

		// Buttons
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-stage-all", theme.AmberStyle.Render("[ STAGE ALL ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-commit", theme.AmberStyle.Render("[ COMMIT ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-push", theme.BaseStyle.Render("[ PUSH ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-pull", theme.BaseStyle.Render("[ PULL ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-fetch", theme.BaseStyle.Render("[ FETCH ]")))
		b.WriteString("  ")
		b.WriteString(zone.Mark("proj-back", theme.DimStyle.Render("[ BACK ]")))
		b.WriteString("\n\n")

		if m.focus == focusCommitMsg {
			b.WriteString(theme.DimStyle.Render("  [Enter] Commit  [Esc] Cancel"))
		} else {
			b.WriteString(theme.DimStyle.Render("  [s] Stage All  [c] Commit  [p] Push  [l] Pull  [f] Fetch  [Esc] Back"))
		}
	}

	if m.focus == focusFileList {
		// Show breadcrumb
		b.WriteString(fmt.Sprintf("  %s %s\n",
			theme.DimStyle.Render("PATH:"),
			theme.BaseStyle.Render(m.currentDir),
		))

		// Git status line
		if m.gitStatus.IsRepo {
			changeCount := len(m.gitStatus.Staged) + len(m.gitStatus.Unstaged) + len(m.gitStatus.Untracked)
			gitLine := fmt.Sprintf("  %s %s", theme.DimStyle.Render("GIT:"), theme.AmberStyle.Render(m.gitStatus.Branch))
			if m.gitStatus.Clean {
				gitLine += "  " + theme.BaseStyle.Render("CLEAN")
			} else {
				gitLine += fmt.Sprintf("  %s", theme.AmberStyle.Render(fmt.Sprintf("%d CHANGES", changeCount)))
			}
			if m.gitStatus.HasRemote && (m.gitStatus.Ahead > 0 || m.gitStatus.Behind > 0) {
				gitLine += fmt.Sprintf("  %s", theme.DimStyle.Render(fmt.Sprintf("[+%d/-%d]", m.gitStatus.Ahead, m.gitStatus.Behind)))
			}
			b.WriteString(gitLine)
			b.WriteString("\n")
		}
		b.WriteString("\n")

		if len(m.files) == 0 {
			b.WriteString(theme.DimStyle.Render("  Empty directory."))
			b.WriteString("\n")
		}

		maxShow := height - 14
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

		b.WriteString("\n  ")
		b.WriteString(zone.Mark("proj-edit", theme.AmberStyle.Render("[ EDIT ]")))
		b.WriteString("  ")
		if m.gitStatus.IsRepo {
			b.WriteString(zone.Mark("proj-git", theme.BaseStyle.Render("[ GIT ]")))
			b.WriteString("  ")
		}
		b.WriteString(zone.Mark("proj-back", theme.BaseStyle.Render("[ BACK ]")))
		b.WriteString("\n\n")
		if m.gitStatus.IsRepo {
			b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Open Dir  [e] Edit File  [g] Git  [Esc] Back"))
		} else {
			b.WriteString(theme.DimStyle.Render("  [j/k] Navigate  [Enter] Open Dir  [e] Edit File  [Esc] Back"))
		}
	}

	return b.String()
}

func (m ProjectsModel) InputFocused() bool {
	return m.focus == focusSetupDir || m.focus == focusSearching || m.focus == focusCommitMsg
}
