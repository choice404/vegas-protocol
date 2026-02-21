package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/choice404/vegas-protocol/vegas-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// --- Tool definitions ---

type Tool struct {
	Name        string
	Command     string
	Category    string
	Description string
	Fields      []ToolField
	BuildArgs   func(fields []ToolField) ([]string, error)
}

type ToolField struct {
	Key         string
	Label       string
	Value       string
	Placeholder string
	Options     []string // If non-empty, cycle through with Enter
}

// --- State machine ---

type itemsState int

const (
	itemsList    itemsState = iota // Tool inventory list
	itemsForm                     // Parameter form
	itemsRunning                  // Command executing
	itemsOutput                   // Scrollable results
)

// --- Tea messages ---

type toolOutputMsg struct {
	output string
	err    error
}

type toolTickMsg time.Time

// --- Model ---

type ItemsModel struct {
	state  itemsState
	tools  []Tool
	cursor int

	// Form (link.go-style cursor + enter-to-edit)
	formCursor  int
	formEditing bool
	formInput   textinput.Model

	// Output
	output        viewport.Model
	outputContent string // raw content (survives viewport resize)
	outputTitle   string

	// Running
	running    bool
	cancelFunc context.CancelFunc
	spinFrame  int
	startTime  time.Time

	// Layout
	width  int
	height int
}

func fieldVal(fields []ToolField, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return strings.TrimSpace(f.Value)
		}
	}
	return ""
}

var toolInventory = []Tool{
	{
		Name:        "Signal Tracker",
		Command:     "dig",
		Category:    "RECON",
		Description: "DNS reconnaissance — resolve domain records",
		Fields: []ToolField{
			{Key: "target", Label: "TARGET", Placeholder: "example.com"},
			{Key: "type", Label: "RECORD TYPE", Value: "A", Options: []string{"A", "AAAA", "MX", "NS", "TXT", "ANY"}},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			target := fieldVal(fields, "target")
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			recType := fieldVal(fields, "type")
			if recType == "" {
				recType = "A"
			}
			return []string{target, recType}, nil
		},
	},
	{
		Name:        "Wasteland Courier",
		Command:     "curl",
		Category:    "NETWORK",
		Description: "HTTP requests — fetch data from the wasteland",
		Fields: []ToolField{
			{Key: "url", Label: "URL", Placeholder: "https://example.com"},
			{Key: "method", Label: "METHOD", Value: "GET", Options: []string{"GET", "POST", "HEAD"}},
			{Key: "headers", Label: "HEADERS", Placeholder: "Key: Value (optional)"},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			url := fieldVal(fields, "url")
			if url == "" {
				return nil, fmt.Errorf("URL is required")
			}
			method := fieldVal(fields, "method")
			if method == "" {
				method = "GET"
			}
			args := []string{"-s", "-S", "-X", method}
			headers := fieldVal(fields, "headers")
			if headers != "" {
				args = append(args, "-H", headers)
			}
			args = append(args, "-i", url)
			return args, nil
		},
	},
	{
		Name:        "Radar Scanner",
		Command:     "nmap",
		Category:    "RECON",
		Description: "Network scanning — map the wasteland's defenses",
		Fields: []ToolField{
			{Key: "target", Label: "TARGET", Placeholder: "host or subnet"},
			{Key: "scan", Label: "SCAN TYPE", Value: "-sT", Options: []string{"-sT", "-sS", "-sP", "-sV"}},
			{Key: "ports", Label: "PORTS", Placeholder: "e.g. 80,443 or 1-1024 (optional)"},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			target := fieldVal(fields, "target")
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			scanType := fieldVal(fields, "scan")
			if scanType == "" {
				scanType = "-sT"
			}
			args := []string{scanType}
			ports := fieldVal(fields, "ports")
			if ports != "" {
				args = append(args, "-p", ports)
			}
			args = append(args, target)
			return args, nil
		},
	},
	{
		Name:        "Intel Lookup",
		Command:     "whois",
		Category:    "RECON",
		Description: "Domain/IP intelligence — who owns this territory?",
		Fields: []ToolField{
			{Key: "target", Label: "TARGET", Placeholder: "domain or IP"},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			target := fieldVal(fields, "target")
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			return []string{target}, nil
		},
	},
	{
		Name:        "Echo Locator",
		Command:     "ping",
		Category:    "NETWORK",
		Description: "Network connectivity — ping the wasteland",
		Fields: []ToolField{
			{Key: "target", Label: "TARGET", Placeholder: "host or IP"},
			{Key: "count", Label: "COUNT", Value: "4", Placeholder: "4"},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			target := fieldVal(fields, "target")
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			count := fieldVal(fields, "count")
			if count == "" {
				count = "4"
			}
			return []string{"-c", count, target}, nil
		},
	},
	{
		Name:        "Route Mapper",
		Command:     "traceroute",
		Category:    "NETWORK",
		Description: "Network path tracing — map the route through the wasteland",
		Fields: []ToolField{
			{Key: "target", Label: "TARGET", Placeholder: "host or IP"},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			target := fieldVal(fields, "target")
			if target == "" {
				return nil, fmt.Errorf("target is required")
			}
			return []string{target}, nil
		},
	},
	{
		Name:        "Socket Scanner",
		Command:     "ss",
		Category:    "NETWORK",
		Description: "Local socket inspection — scan open connections",
		Fields: []ToolField{
			{Key: "filter", Label: "FILTER", Value: "listening", Options: []string{"all", "listening", "tcp", "udp"}},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			filter := fieldVal(fields, "filter")
			switch filter {
			case "all":
				return []string{"-a", "-n"}, nil
			case "listening":
				return []string{"-l", "-n"}, nil
			case "tcp":
				return []string{"-t", "-n"}, nil
			case "udp":
				return []string{"-u", "-n"}, nil
			default:
				return []string{"-l", "-n"}, nil
			}
		},
	},
	{
		Name:        "Cipher Kit",
		Command:     "openssl",
		Category:    "CRYPTO",
		Description: "Hashing utility — encrypt data for the wasteland",
		Fields: []ToolField{
			{Key: "input", Label: "INPUT TEXT", Placeholder: "text to hash"},
			{Key: "algo", Label: "ALGORITHM", Value: "sha256", Options: []string{"md5", "sha1", "sha256", "sha512"}},
		},
		BuildArgs: func(fields []ToolField) ([]string, error) {
			input := fieldVal(fields, "input")
			if input == "" {
				return nil, fmt.Errorf("input text is required")
			}
			algo := fieldVal(fields, "algo")
			if algo == "" {
				algo = "sha256"
			}
			// openssl dgst uses printf input via pipe, handled specially
			return []string{algo, input}, nil
		},
	},
}

func NewItemsModel() ItemsModel {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 40

	return ItemsModel{
		state: itemsList,
		tools: toolInventory,
		formInput: ti,
	}
}

func (m ItemsModel) Init() tea.Cmd {
	return nil
}

// InputFocused returns true when the form text input has focus.
func (m ItemsModel) InputFocused() bool {
	return m.formEditing
}

func (m ItemsModel) Update(msg tea.Msg) (ItemsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case toolOutputMsg:
		m.running = false
		m.cancelFunc = nil
		m.state = itemsOutput
		if msg.err != nil {
			if msg.output != "" {
				m.outputContent = msg.output + "\n\n" + theme.RedStyle.Render("ERROR: "+msg.err.Error())
			} else {
				m.outputContent = theme.RedStyle.Render("ERROR: " + msg.err.Error())
			}
		} else {
			m.outputContent = msg.output
		}
		m.output.SetContent(m.outputContent)
		m.output.GotoTop()
		return m, nil

	case toolTickMsg:
		if m.state != itemsRunning {
			return m, nil
		}
		m.spinFrame++
		return m, m.tickCmd()

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			cmd := m.handleClick(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
		// Wheel scroll
		switch m.state {
		case itemsList:
			if msg.Button == tea.MouseButtonWheelUp && m.cursor > 0 {
				m.cursor--
			}
			if msg.Button == tea.MouseButtonWheelDown && m.cursor < len(m.tools)-1 {
				m.cursor++
			}
		case itemsForm:
			tool := m.tools[m.cursor]
			if msg.Button == tea.MouseButtonWheelUp && m.formCursor > 0 {
				m.formCursor--
			}
			if msg.Button == tea.MouseButtonWheelDown && m.formCursor < len(tool.Fields)-1 {
				m.formCursor++
			}
		case itemsOutput:
			var cmd tea.Cmd
			m.output, cmd = m.output.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// Forward non-key/mouse messages (blink cursor, etc.)
	if m.formEditing {
		var cmd tea.Cmd
		m.formInput, cmd = m.formInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == itemsOutput {
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ItemsModel) handleClick(msg tea.MouseMsg) tea.Cmd {
	switch m.state {
	case itemsList:
		if zone.Get("items-use").InBounds(msg) {
			return m.enterForm()
		}
		for i := range m.tools {
			if zone.Get(fmt.Sprintf("tool-%d", i)).InBounds(msg) {
				if m.cursor == i {
					return m.enterForm()
				}
				m.cursor = i
				return nil
			}
		}

	case itemsForm:
		if zone.Get("items-run").InBounds(msg) {
			return m.runTool()
		}
		if zone.Get("items-form-back").InBounds(msg) {
			m.state = itemsList
			m.formEditing = false
			m.formInput.Blur()
			return nil
		}
		tool := m.tools[m.cursor]
		for i := range tool.Fields {
			if zone.Get(fmt.Sprintf("items-field-%d", i)).InBounds(msg) {
				m.formCursor = i
				return nil
			}
		}

	case itemsRunning:
		if zone.Get("items-cancel").InBounds(msg) {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return nil
		}

	case itemsOutput:
		if zone.Get("items-run-again").InBounds(msg) {
			return m.runTool()
		}
		if zone.Get("items-output-back").InBounds(msg) {
			m.state = itemsForm
			return nil
		}
	}
	return nil
}

func (m *ItemsModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch m.state {
	case itemsList:
		return m.handleListKey(msg)
	case itemsForm:
		return m.handleFormKey(msg)
	case itemsRunning:
		if msg.String() == "esc" && m.cancelFunc != nil {
			m.cancelFunc()
		}
		return nil
	case itemsOutput:
		return m.handleOutputKey(msg)
	}
	return nil
}

func (m *ItemsModel) handleListKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tools)-1 {
			m.cursor++
		}
	case "enter":
		return m.enterForm()
	}
	return nil
}

func (m *ItemsModel) handleFormKey(msg tea.KeyMsg) tea.Cmd {
	if m.formEditing {
		switch msg.String() {
		case "enter":
			val := m.formInput.Value()
			m.tools[m.cursor].Fields[m.formCursor].Value = val
			m.formEditing = false
			m.formInput.Blur()
			return nil
		case "esc":
			m.formEditing = false
			m.formInput.Blur()
			return nil
		}
		var cmd tea.Cmd
		m.formInput, cmd = m.formInput.Update(msg)
		return cmd
	}

	tool := m.tools[m.cursor]
	switch msg.String() {
	case "up", "k":
		if m.formCursor > 0 {
			m.formCursor--
		}
	case "down", "j":
		if m.formCursor < len(tool.Fields)-1 {
			m.formCursor++
		}
	case "enter":
		field := tool.Fields[m.formCursor]
		if len(field.Options) > 0 {
			// Cycle through options
			current := field.Value
			idx := 0
			for i, opt := range field.Options {
				if opt == current {
					idx = (i + 1) % len(field.Options)
					break
				}
			}
			m.tools[m.cursor].Fields[m.formCursor].Value = field.Options[idx]
			return nil
		}
		// Open text input for editing
		m.formEditing = true
		m.formInput.SetValue(field.Value)
		m.formInput.Focus()
		return textinput.Blink
	case "tab":
		return m.runTool()
	case "esc":
		m.state = itemsList
		m.formEditing = false
		m.formInput.Blur()
		return nil
	}
	return nil
}

func (m *ItemsModel) handleOutputKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.state = itemsList
		return nil
	case "b":
		m.state = itemsForm
		return nil
	}
	// Forward scroll keys to viewport
	var cmd tea.Cmd
	m.output, cmd = m.output.Update(msg)
	return cmd
}

func (m *ItemsModel) enterForm() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.tools) {
		return nil
	}
	m.state = itemsForm
	m.formCursor = 0
	m.formEditing = false
	return nil
}

func (m *ItemsModel) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return toolTickMsg(t)
	})
}

func (m *ItemsModel) runTool() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.tools) {
		return nil
	}
	tool := m.tools[m.cursor]

	m.state = itemsRunning
	m.running = true
	m.spinFrame = 0
	m.startTime = time.Now()

	// Build the output title
	args, err := tool.BuildArgs(tool.Fields)
	if err != nil {
		return func() tea.Msg {
			return toolOutputMsg{err: err}
		}
	}
	m.outputTitle = fmt.Sprintf("%s > %s %s", tool.Name, tool.Command, strings.Join(args, " "))

	return tea.Batch(m.runToolCmd(tool), m.tickCmd())
}

func (m *ItemsModel) runToolCmd(tool Tool) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	m.cancelFunc = cancel

	return func() tea.Msg {
		defer cancel()

		args, err := tool.BuildArgs(tool.Fields)
		if err != nil {
			return toolOutputMsg{err: err}
		}

		// Special handling for openssl: pipe input text
		if tool.Command == "openssl" && len(args) >= 2 {
			algo := args[0]
			input := args[1]
			cmd := exec.CommandContext(ctx, "sh", "-c",
				fmt.Sprintf("printf '%%s' %q | openssl dgst -%s", input, algo))
			out, cmdErr := cmd.CombinedOutput()
			return toolOutputMsg{output: string(out), err: cmdErr}
		}

		cmd := exec.CommandContext(ctx, tool.Command, args...)
		out, cmdErr := cmd.CombinedOutput()
		return toolOutputMsg{output: string(out), err: cmdErr}
	}
}

// --- Views ---

func (m ItemsModel) View(width, height int) string {
	if width != m.width || height != m.height {
		m.width = width
		m.height = height
	}

	switch m.state {
	case itemsList:
		return m.viewList(width, height)
	case itemsForm:
		return m.viewForm(width, height)
	case itemsRunning:
		return m.viewRunning(width, height)
	case itemsOutput:
		return m.viewOutput(width, height)
	}
	return ""
}

func (m *ItemsModel) viewList(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" TOOLS INVENTORY "))
	b.WriteString("\n\n")

	for i, tool := range m.tools {
		cursor := "  "
		if i == m.cursor {
			cursor = theme.AmberStyle.Render("> ")
		}

		nameStyle := theme.BaseStyle
		if i == m.cursor {
			nameStyle = theme.AmberStyle
		}

		cmdTag := theme.DimStyle.Render(fmt.Sprintf("(%s)", tool.Command))
		catTag := theme.DimStyle.Render(fmt.Sprintf("[%s]", tool.Category))
		line := fmt.Sprintf(" %s%s %s  %s", cursor, nameStyle.Render(tool.Name), cmdTag, catTag)
		line = zone.Mark(fmt.Sprintf("tool-%d", i), line)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Detail panel for selected tool
	if m.cursor >= 0 && m.cursor < len(m.tools) {
		tool := m.tools[m.cursor]
		detailWidth := width - 6
		if detailWidth < 30 {
			detailWidth = 30
		}
		if detailWidth > 60 {
			detailWidth = 60
		}

		b.WriteString("\n")
		b.WriteString(theme.DimStyle.Render(" " + strings.Repeat("─", detailWidth)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(" %s\n", theme.BoldStyle.Render(tool.Name)))
		b.WriteString(fmt.Sprintf(" %s\n", theme.BaseStyle.Render(tool.Description)))
	}

	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("items-use", theme.AmberStyle.Render("[ USE ]")))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render(" [j/k] Navigate  [Enter/Click] Use Tool"))

	return b.String()
}

func (m *ItemsModel) viewForm(width, height int) string {
	if m.cursor < 0 || m.cursor >= len(m.tools) {
		return ""
	}
	tool := m.tools[m.cursor]

	var b strings.Builder

	header := fmt.Sprintf(" %s (%s) — %s ", strings.ToUpper(tool.Name), tool.Command, tool.Category)
	b.WriteString(theme.TitleStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" %s\n", theme.BaseStyle.Render(tool.Description)))
	detailWidth := width - 6
	if detailWidth < 30 {
		detailWidth = 30
	}
	if detailWidth > 60 {
		detailWidth = 60
	}
	b.WriteString(theme.DimStyle.Render(" " + strings.Repeat("─", detailWidth)))
	b.WriteString("\n\n")

	labelWidth := 14
	for i, field := range tool.Fields {
		cursor := "  "
		if i == m.formCursor {
			cursor = theme.AmberStyle.Render("> ")
		}

		label := theme.DimStyle.Render(fmt.Sprintf("%-*s", labelWidth, field.Label))

		var val string
		if m.formEditing && i == m.formCursor {
			val = m.formInput.View()
		} else {
			displayVal := field.Value
			if displayVal == "" {
				if field.Placeholder != "" {
					displayVal = field.Placeholder
				} else {
					displayVal = "(empty)"
				}
			}
			valStyle := theme.BaseStyle
			if i == m.formCursor {
				valStyle = theme.AmberStyle
			}
			if field.Value == "" {
				valStyle = theme.DimStyle
			}
			if len(field.Options) > 0 && i == m.formCursor {
				// Show cycle indicator
				displayVal = displayVal + " ↵"
			}
			val = valStyle.Render(displayVal)
		}

		line := fmt.Sprintf("  %s%s %s", cursor, label, val)
		if !m.formEditing || i != m.formCursor {
			line = zone.Mark(fmt.Sprintf("items-field-%d", i), line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(zone.Mark("items-run", theme.AmberStyle.Render("[ RUN ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("items-form-back", theme.BaseStyle.Render("[ BACK ]")))
	b.WriteString("\n\n")

	if m.formEditing {
		b.WriteString(theme.DimStyle.Render(" [Enter] Save  [Esc] Cancel"))
	} else {
		b.WriteString(theme.DimStyle.Render(" [j/k] Navigate  [Enter] Edit/Cycle  [Tab] Run  [Esc] Back"))
	}

	return b.String()
}

func (m *ItemsModel) viewRunning(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.TitleStyle.Render(" EXECUTING "))
	b.WriteString("\n\n")

	if m.cursor >= 0 && m.cursor < len(m.tools) {
		tool := m.tools[m.cursor]
		b.WriteString(theme.AmberStyle.Render(fmt.Sprintf(" %s (%s)", tool.Name, tool.Command)))
		b.WriteString("\n\n")
	}

	// Animated spinner
	spinChars := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spin := spinChars[m.spinFrame%len(spinChars)]
	elapsed := time.Since(m.startTime).Truncate(time.Second)
	b.WriteString(theme.AmberStyle.Render(fmt.Sprintf(" %s EXECUTING COMMAND...", spin)))
	b.WriteString(theme.DimStyle.Render(fmt.Sprintf("  [%s elapsed]", elapsed)))
	b.WriteString("\n\n")

	if m.outputTitle != "" {
		b.WriteString(theme.DimStyle.Render(fmt.Sprintf(" %s", m.outputTitle)))
		b.WriteString("\n\n")
	}

	b.WriteString(theme.DimStyle.Render(" Timeout: 30 seconds"))
	b.WriteString("\n\n")

	b.WriteString("  ")
	b.WriteString(zone.Mark("items-cancel", theme.RedStyle.Render("[ CANCEL ]")))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render(" [Esc] Cancel"))

	return b.String()
}

func (m *ItemsModel) viewOutput(width, height int) string {
	var b strings.Builder

	title := m.outputTitle
	if title == "" {
		title = "OUTPUT"
	}
	b.WriteString(theme.TitleStyle.Render(fmt.Sprintf(" %s ", title)))
	b.WriteString("\n")

	divWidth := width - 4
	if divWidth < 10 {
		divWidth = 10
	}
	if divWidth > 80 {
		divWidth = 80
	}
	b.WriteString(theme.DimStyle.Render(" " + strings.Repeat("─", divWidth)))
	b.WriteString("\n\n")

	vpHeight := height - 10
	if vpHeight < 5 {
		vpHeight = 5
	}
	vpWidth := width - 2
	if vpWidth < 20 {
		vpWidth = 20
	}

	if m.output.Width != vpWidth || m.output.Height != vpHeight {
		m.output = viewport.New(vpWidth, vpHeight)
		if m.outputContent != "" {
			m.output.SetContent(m.outputContent)
		}
	}

	b.WriteString(m.output.View())
	b.WriteString("\n\n")

	b.WriteString("  ")
	b.WriteString(zone.Mark("items-run-again", theme.AmberStyle.Render("[ RUN AGAIN ]")))
	b.WriteString("  ")
	b.WriteString(zone.Mark("items-output-back", theme.BaseStyle.Render("[ BACK ]")))
	b.WriteString("\n\n")
	b.WriteString(theme.DimStyle.Render(" [j/k/PgUp/PgDn] Scroll  [B] Back to form  [Esc] Back to list"))

	return b.String()
}

// wrapText wraps text to the given width. Used by items and data tabs.
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
