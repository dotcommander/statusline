package main

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Tokyo Night palette ──────────────────────────────────────────────────

var (
	clrMuted      = lipgloss.Color("#565f89")
	clrSoft       = lipgloss.Color("#a9b1d6")
	clrActive     = lipgloss.Color("#c0caf5")
	clrSeparator  = lipgloss.Color("#3b4261")
	clrGit        = lipgloss.Color("#7aa2f7")
	clrGreen      = lipgloss.Color("#9ece6a")
	clrPaleOrange = lipgloss.Color("#d4a373")
	clrBg         = lipgloss.Color("#1a1b26")
)

// ─── Lipgloss styles ─────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrActive).
			MarginBottom(1)

	tabStyle = lipgloss.NewStyle().
			Foreground(clrMuted).
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(clrActive).
			Bold(true).
			Padding(0, 2).
			Underline(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(clrSoft).
			Width(16)

	valueStyle = lipgloss.NewStyle().
			Foreground(clrActive)

	emptyValueStyle = lipgloss.NewStyle().
			Foreground(clrMuted).
			Italic(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(clrBg).
			Background(clrGit).
			Bold(true)

	previewBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrSeparator).
			Padding(0, 1).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(clrGreen)

	helpStyle = lipgloss.NewStyle().
			Foreground(clrMuted).
			MarginTop(1)
)

// ─── Sections ─────────────────────────────────────────────────────────────

type section int

const (
	secLayout section = iota
	secAppearance
	secPrompts
	secContext
	secTokens
	secCount
)

var sectionNames = []string{"Layout", "Appearance", "Prompts", "Context", "Tokens"}

// ─── Field definitions ────────────────────────────────────────────────────

type fieldDef struct {
	label string
	get   func(Config) string
	set   func(*Config, string)
}

var knownTokens = []string{"dir", "git", "project", "model", "ctx", "label", "dc", "prompts"}

func fieldsForSection(sec section) []fieldDef {
	switch sec {
	case secLayout:
		return []fieldDef{
			{"Line 1", func(c Config) string { return c.Line1 }, func(c *Config, v string) { c.Line1 = v }},
			{"Line 2", func(c Config) string { return c.Line2 }, func(c *Config, v string) { c.Line2 = v }},
		}
	case secAppearance:
		return []fieldDef{
			{"Separator", func(c Config) string { return c.Separator }, func(c *Config, v string) { c.Separator = v }},
			{"Dot", func(c Config) string { return c.Dot }, func(c *Config, v string) { c.Dot = v }},
		}
	case secPrompts:
		return []fieldDef{
			{"Max prompts", func(c Config) string { return itoa(c.Prompts.Max) }, func(c *Config, v string) { c.Prompts.Max = atoi(v) }},
			{"Newest words", func(c Config) string { return itoa(c.Prompts.NewestWords) }, func(c *Config, v string) { c.Prompts.NewestWords = atoi(v) }},
			{"Older words", func(c Config) string { return itoa(c.Prompts.OlderWords) }, func(c *Config, v string) { c.Prompts.OlderWords = atoi(v) }},
			{"Cache TTL (s)", func(c Config) string { return itoa(c.Prompts.CacheTTL) }, func(c *Config, v string) { c.Prompts.CacheTTL = atoi(v) }},
		}
	case secContext:
		return []fieldDef{
			{"Warning %", func(c Config) string { return itoa(c.Context.WarningPct) }, func(c *Config, v string) { c.Context.WarningPct = atoi(v) }},
			{"Critical %", func(c Config) string { return itoa(c.Context.CriticalPct) }, func(c *Config, v string) { c.Context.CriticalPct = atoi(v) }},
			{"Alert %", func(c Config) string { return itoa(c.Context.AlertPct) }, func(c *Config, v string) { c.Context.AlertPct = atoi(v) }},
		}
	case secTokens:
		return tokenFields()
	}
	return nil
}

func tokenFields() []fieldDef {
	var fields []fieldDef
	for _, name := range knownTokens {
		n := name
		fields = append(fields, fieldDef{
			label: n + " style",
			get: func(c Config) string {
				if tc := c.Tokens[n]; tc != nil {
					return tc.Style
				}
				return ""
			},
			set: func(c *Config, v string) {
				ensureToken(c, n)
				c.Tokens[n].Style = v
			},
		})
		fields = append(fields, fieldDef{
			label: n + " max_length",
			get: func(c Config) string {
				if tc := c.Tokens[n]; tc != nil && tc.MaxLength > 0 {
					return itoa(tc.MaxLength)
				}
				return ""
			},
			set: func(c *Config, v string) {
				ensureToken(c, n)
				c.Tokens[n].MaxLength = atoi(v)
			},
		})
	}
	return fields
}

func ensureToken(c *Config, name string) {
	if c.Tokens == nil {
		c.Tokens = make(map[string]*TokenConfig)
	}
	if c.Tokens[name] == nil {
		c.Tokens[name] = &TokenConfig{}
	}
}

// ─── Model ────────────────────────────────────────────────────────────────

type model struct {
	cfg        Config
	configPath string

	section section
	cursor  int
	editing bool
	input   textinput.Model

	width   int
	height  int
	message string
}

func newModel(cfg Config, path string) model {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.Prompt = "  > "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(clrGit)
	ti.TextStyle = lipgloss.NewStyle().Foreground(clrActive)
	return model{
		cfg:        cfg,
		configPath: path,
		input:      ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// ─── Update ───────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.editing {
			return m.updateEditing(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m model) updateEditing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		fields := fieldsForSection(m.section)
		if m.cursor < len(fields) {
			fields[m.cursor].set(&m.cfg, m.input.Value())
		}
		m.editing = false
		m.input.Blur()
		m.message = ""
		return m, nil

	case tea.KeyEsc:
		m.editing = false
		m.input.Blur()
		m.message = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fields := fieldsForSection(m.section)

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab", "right", "l":
		m.section = (m.section + 1) % secCount
		m.cursor = 0
		m.message = ""
		return m, nil

	case "shift+tab", "left", "h":
		m.section = (m.section - 1 + secCount) % secCount
		m.cursor = 0
		m.message = ""
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(fields)-1 {
			m.cursor++
		}
		return m, nil

	case "enter":
		if m.cursor < len(fields) {
			m.editing = true
			m.input.SetValue(fields[m.cursor].get(m.cfg))
			m.input.Focus()
			m.input.CursorEnd()
			return m, textinput.Blink
		}

	case "s", "ctrl+s":
		if err := saveConfig(m.configPath, m.cfg); err != nil {
			m.message = "Error: " + err.Error()
		} else {
			m.message = "Saved to " + m.configPath
		}
		return m, nil

	case "r":
		m.cfg = defaultConfig
		m.message = "Reset to defaults (not saved)"
		return m, nil
	}

	return m, nil
}

// ─── View ─────────────────────────────────────────────────────────────────

func (m model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Statusline Configurator"))
	b.WriteString("\n")

	// Preview
	preview := renderPreview(m.cfg, m.width-4)
	b.WriteString(previewBoxStyle.Render(preview))
	b.WriteString("\n")

	// Tab bar
	var tabs []string
	for i, name := range sectionNames {
		if section(i) == m.section {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	b.WriteString("\n\n")

	// Fields
	fields := fieldsForSection(m.section)
	for i, f := range fields {
		val := f.get(m.cfg)

		if m.editing && i == m.cursor {
			b.WriteString(cursorStyle.Render(" " + f.label + " "))
			b.WriteString("\n")
			b.WriteString(m.input.View())
			b.WriteString("\n")
			continue
		}

		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render(">") + " "
		}

		label := labelStyle.Render(f.label)
		var rendered string
		if val == "" {
			rendered = emptyValueStyle.Render("—")
		} else {
			rendered = valueStyle.Render(val)
		}

		b.WriteString(prefix + label + rendered + "\n")
	}

	// Status message
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(m.message))
	}

	// Help
	help := "tab/shift+tab: section  j/k: navigate  enter: edit  s: save  r: reset  q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

// ─── Preview renderer ─────────────────────────────────────────────────────

var tokenRe = regexp.MustCompile(`\[(\w+)\]`)

func parseTokens(tmpl string) []string {
	matches := tokenRe.FindAllStringSubmatch(tmpl, -1)
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m[1]
	}
	return out
}

var mockTokenData = map[string]struct {
	text  string
	color lipgloss.Color
}{
	"dir":     {"statusline", clrGit},
	"git":     {"\ue0a0 main", clrGreen},
	"project": {"go 1.26", clrSoft},
	"model":   {"claude opus 4.6", clrPaleOrange},
	"ctx":     {"ctx:42k", clrGreen},
	"label":   {"cc 1.0.33", clrPaleOrange},
	"dc":      {"dc:1.6.1", clrPaleOrange},
	"prompts": {"› review work done", clrMuted},
}

func renderPreview(cfg Config, maxWidth int) string {
	sep := lipgloss.NewStyle().Foreground(clrSeparator).Render(cfg.Separator)
	dot := lipgloss.NewStyle().Foreground(clrGreen).Render(cfg.Dot)

	renderLine := func(tmpl string, showDot bool) string {
		tokens := parseTokens(tmpl)
		var parts []string
		for _, name := range tokens {
			mock, ok := mockTokenData[name]
			if !ok {
				continue
			}
			style := lipgloss.NewStyle().Foreground(mock.color)
			// Apply per-token style override in preview
			if tc := cfg.Tokens[name]; tc != nil && tc.Style != "" {
				style = parsePreviewStyle(tc.Style)
			}
			text := mock.text
			// Apply max_length for git
			if name == "git" {
				if tc := cfg.Tokens["git"]; tc != nil && tc.MaxLength > 0 {
					// Branch is everything after the icon+space
					branchIdx := strings.Index(text, " ")
					if branchIdx >= 0 {
						branch := text[branchIdx+1:]
						runes := []rune(branch)
						if len(runes) > tc.MaxLength {
							if tc.MaxLength > 3 {
								branch = string(runes[:tc.MaxLength-3]) + "..."
							} else {
								branch = string(runes[:tc.MaxLength])
							}
						}
						text = text[:branchIdx+1] + branch
					}
				}
			}
			parts = append(parts, style.Render(text))
		}
		line := strings.Join(parts, sep)
		if showDot {
			line = dot + " " + line
		}
		return line
	}

	line1 := renderLine(cfg.Line1, true)
	line2 := renderLine(cfg.Line2, false)
	return line1 + "\n" + line2
}

func parsePreviewStyle(s string) lipgloss.Style {
	style := lipgloss.NewStyle()
	for _, token := range strings.Fields(s) {
		switch strings.ToLower(token) {
		case "bold":
			style = style.Bold(true)
		case "italic":
			style = style.Italic(true)
		case "underline":
			style = style.Underline(true)
		case "dim":
			style = style.Faint(true)
		case "strikethrough":
			style = style.Strikethrough(true)
		default:
			if strings.HasPrefix(token, "#") && len(token) == 7 {
				style = style.Foreground(lipgloss.Color(token))
			}
		}
	}
	return style
}

// ─── Helpers ──────────────────────────────────────────────────────────────

func itoa(n int) string { return strconv.Itoa(n) }

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
