package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/jiriki-labs/agentic-wallet/internal/config"
	"github.com/jiriki-labs/agentic-wallet/internal/policy"
)

const (
	policyFieldCount = 7
)

type policyScreen int

const (
	policyScreenMain policyScreen = iota
	policyScreenString
	policyScreenList
)

type policyTUIModel struct {
	path   string
	cfg    policy.Config
	dirty  bool
	screen policyScreen

	width  int
	height int
	cursor int

	status string
	errMsg string

	// string editor
	editField  int
	editLabel  string
	editBuffer string

	// list editor
	listField int
	listLabel string
	listItems []string
	listCur   int
	listAdd   bool
}

func runPolicyTUI() error {
	path := policyFileFlag
	if path == "" {
		path = config.PolicyFile()
	}
	cfg := policy.DefaultConfig()
	if _, err := os.Stat(path); err == nil {
		loaded, err := policy.ReadConfig(path)
		if err != nil {
			return err
		}
		cfg = loaded
	}

	m := policyTUIModel{
		path: path,
		cfg:  cfg,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("policy tui: %w", err)
	}
	return nil
}

func (m policyTUIModel) Init() tea.Cmd { return nil }

func (m policyTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch m.screen {
		case policyScreenMain:
			return m.updateMain(msg)
		case policyScreenString:
			return m.updateString(msg)
		case policyScreenList:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m policyTUIModel) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.dirty && !strings.HasPrefix(m.errMsg, "unsaved") {
			m.errMsg = "unsaved changes — press q again to discard, or s to save"
			return m, nil
		}
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < policyFieldCount-1 {
			m.cursor++
		}
	case "s":
		if err := policy.Save(m.path, m.cfg); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.dirty = false
		m.status = "saved " + m.path
		return m, nil
	case "enter":
		return m.beginEdit()
	}
	return m, nil
}

func (m policyTUIModel) beginEdit() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		modes := []string{"dry-run", "confirm", "auto"}
		idx := 0
		for i, mode := range modes {
			if mode == m.cfg.Mode {
				idx = i
				break
			}
		}
		m.cfg.Mode = modes[(idx+1)%len(modes)]
		m.dirty = true
		m.status = "mode → " + m.cfg.Mode
		return m, nil
	case 1:
		m.screen = policyScreenString
		m.editField = 1
		m.editLabel = "Max amount per request"
		m.editBuffer = m.cfg.MaxAmountPerRequest
	case 2:
		m.screen = policyScreenString
		m.editField = 2
		m.editLabel = "Daily limit"
		m.editBuffer = m.cfg.DailyLimit
	case 3:
		m.screen = policyScreenString
		m.editField = 3
		m.editLabel = "Require approval above"
		m.editBuffer = m.cfg.RequireApprovalAbove
	case 4:
		m.openList(4, "Allowed tokens", m.cfg.AllowedTokens)
	case 5:
		m.openList(5, "Allowed chains", m.cfg.AllowedChains)
	case 6:
		m.openList(6, "Allowed merchants", m.cfg.AllowedMerchants)
	}
	return m, nil
}

func (m *policyTUIModel) openList(field int, label string, items []string) {
	m.screen = policyScreenList
	m.listField = field
	m.listLabel = label
	m.listItems = append([]string(nil), items...)
	m.listCur = 0
	m.listAdd = false
}

func (m *policyTUIModel) applyStringEdit(val string) {
	switch m.editField {
	case 1:
		m.cfg.MaxAmountPerRequest = val
	case 2:
		m.cfg.DailyLimit = val
	case 3:
		m.cfg.RequireApprovalAbove = val
	}
}

func (m *policyTUIModel) applyListEdit(items []string) {
	switch m.listField {
	case 4:
		m.cfg.AllowedTokens = items
	case 5:
		m.cfg.AllowedChains = items
	case 6:
		m.cfg.AllowedMerchants = items
	}
}

func (m policyTUIModel) updateString(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = policyScreenMain
		return m, nil
	case "enter":
		m.applyStringEdit(m.editBuffer)
		m.dirty = true
		m.screen = policyScreenMain
		m.status = m.editLabel + " updated"
		return m, nil
	case "backspace", "ctrl+h":
		if len(m.editBuffer) > 0 {
			m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.editBuffer += string(msg.Runes)
		}
	}
	return m, nil
}

func (m policyTUIModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = policyScreenMain
		return m, nil
	case "enter":
		if m.listAdd {
			val := strings.TrimSpace(m.editBuffer)
			if val != "" {
				m.listItems = append(m.listItems, val)
				m.dirty = true
			}
			m.listAdd = false
			m.editBuffer = ""
			m.listCur = len(m.listItems) - 1
			return m, nil
		}
		m.applyListEdit(m.listItems)
		m.dirty = true
		m.screen = policyScreenMain
		m.status = m.listLabel + " updated"
		return m, nil
	case "up", "k":
		if m.listAdd {
			m.listAdd = false
			m.editBuffer = ""
		} else if m.listCur > 0 {
			m.listCur--
		}
	case "down", "j":
		if m.listCur < len(m.listItems) {
			m.listCur++
		}
	case "a":
		m.listAdd = true
		m.editBuffer = ""
		m.listCur = len(m.listItems)
	case "d":
		if len(m.listItems) == 0 || m.listAdd {
			return m, nil
		}
		idx := m.listCur
		if idx >= len(m.listItems) {
			idx = len(m.listItems) - 1
		}
		m.listItems = append(m.listItems[:idx], m.listItems[idx+1:]...)
		if m.listCur >= len(m.listItems) && len(m.listItems) > 0 {
			m.listCur = len(m.listItems) - 1
		}
		m.dirty = true
	default:
		if m.listAdd && len(msg.Runes) > 0 {
			m.editBuffer += string(msg.Runes)
		} else if m.listAdd && msg.String() == "backspace" && len(m.editBuffer) > 0 {
			m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
		}
	}
	return m, nil
}

func (m policyTUIModel) View() string {
	if m.width <= 0 {
		m.width = 80
	}
	switch m.screen {
	case policyScreenString:
		return m.viewString()
	case policyScreenList:
		return m.viewList()
	default:
		return m.viewMain()
	}
}

func (m policyTUIModel) viewMain() string {
	var b strings.Builder
	b.WriteString(tuiTitleStyle.Render(" Policy "))
	b.WriteString(tuiDimStyle.Render(" · "))
	b.WriteString(tuiDescStyle.Render(m.path))
	if m.dirty {
		b.WriteString(tuiDimStyle.Render(" · "))
		b.WriteString(tuiSelStyle.Render("unsaved"))
	}
	b.WriteString("\n\n")

	rows := []struct {
		label string
		value string
	}{
		{"Mode", m.cfg.Mode},
		{"Max per request", m.cfg.MaxAmountPerRequest},
		{"Daily limit", m.cfg.DailyLimit},
		{"Require approval above", m.cfg.RequireApprovalAbove},
		{"Allowed tokens", formatListPreview(m.cfg.AllowedTokens)},
		{"Allowed chains", formatListPreview(m.cfg.AllowedChains)},
		{"Allowed merchants", formatListPreview(m.cfg.AllowedMerchants)},
	}
	for i, row := range rows {
		line := fmt.Sprintf("%-24s %s", row.label, row.value)
		if i == m.cursor {
			b.WriteString(tuiSelStyle.Render("› " + line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.errMsg != "" {
		b.WriteString(tuiSelStyle.Render("! " + m.errMsg))
		b.WriteString("\n")
	} else if m.status != "" {
		b.WriteString(tuiDescStyle.Render(m.status))
		b.WriteString("\n")
	}
	footer := tuiDimStyle.Render("↑/↓ navigate  ·  enter edit/cycle mode  ·  s save  ·  q quit")
	b.WriteString(footer)
	return tuiBorder.Width(m.width - 4).Render(b.String())
}

func (m policyTUIModel) viewString() string {
	var b strings.Builder
	b.WriteString(tuiTitleStyle.Render(" " + m.editLabel + " "))
	b.WriteString("\n\n")
	b.WriteString(tuiSelStyle.Render("› "))
	b.WriteString(m.editBuffer)
	b.WriteString(tuiDimStyle.Render("█"))
	b.WriteString("\n\n")
	b.WriteString(tuiDimStyle.Render("enter confirm  ·  esc cancel"))
	return tuiBorder.Width(m.width - 4).Render(b.String())
}

func (m policyTUIModel) viewList() string {
	var b strings.Builder
	b.WriteString(tuiTitleStyle.Render(" " + m.listLabel + " "))
	b.WriteString("\n\n")
	for i, item := range m.listItems {
		prefix := "  "
		if i == m.listCur && !m.listAdd {
			prefix = tuiSelStyle.Render("› ")
		}
		b.WriteString(prefix + item + "\n")
	}
	if m.listAdd {
		b.WriteString(tuiSelStyle.Render("› "))
		b.WriteString(m.editBuffer)
		b.WriteString(tuiDimStyle.Render("█"))
		b.WriteString("\n")
	} else if m.listCur == len(m.listItems) {
		b.WriteString(tuiDimStyle.Render("› (new item — press a)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(tuiDimStyle.Render("↑/↓  ·  a add  ·  d delete  ·  enter apply  ·  esc back"))
	return tuiBorder.Width(m.width - 4).Render(b.String())
}

func formatListPreview(items []string) string {
	if len(items) == 0 {
		return "(empty)"
	}
	if len(items) <= 2 {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:2], ", ") + fmt.Sprintf(" (+%d)", len(items)-2)
}
