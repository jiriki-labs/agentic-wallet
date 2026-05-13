package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI to run common commands",
	Long:  "Full-screen keyboard menu. Choose an action and Jiriki re-execs the matching CLI command with your terminal restored.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

type menuItem struct {
	title    string
	subtitle string
	argv     []string // nil: exit TUI only; non-nil: exec os.Args[0] with argv
}

type tuiMenuModel struct {
	cursor  int
	width   int
	height  int
	choices []menuItem
}

func (m tuiMenuModel) Init() tea.Cmd {
	return nil
}

func (m tuiMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return tuiPickedModel{argv: nil}, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			return tuiPickedModel{argv: m.choices[m.cursor].argv}, tea.Quit
		}
	}
	return m, nil
}

var (
	tuiTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	tuiDescStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiSelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	tuiDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	tuiBorder     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("238"))
)

func (m tuiMenuModel) View() string {
	if m.width <= 0 {
		m.width = 80
	}
	var b strings.Builder
	b.WriteString(tuiTitleStyle.Render(" Jiriki "))
	b.WriteString(tuiDimStyle.Render(" · "))
	b.WriteString(tuiDescStyle.Render("wallet daemon"))
	b.WriteString("\n\n")

	for i, c := range m.choices {
		var line string
		if i == m.cursor {
			line = tuiSelStyle.Render("› " + c.title)
			if c.subtitle != "" {
				line += "\n" + tuiDescStyle.Render("  "+c.subtitle)
			}
		} else {
			line = c.title
			if c.subtitle != "" {
				line += "\n" + tuiDescStyle.Render(c.subtitle)
			}
		}
		b.WriteString(line)
		b.WriteString("\n\n")
	}

	footer := tuiDimStyle.Render("↑/k · ↓/j navigate  ·  enter run  ·  q esc quit")
	b.WriteString(footer)
	return tuiBorder.Width(m.width - 4).Render(b.String())
}

type tuiPickedModel struct {
	argv []string
}

func (tuiPickedModel) Init() tea.Cmd { return nil }

func (m tuiPickedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m tuiPickedModel) View() string { return "" }

func runTUI() error {
	choices := []menuItem{
		{"Init wallet", "Create keystore and print address", []string{"init"}},
		{"Start daemon", "Unlock, load policy, listen for API", []string{"up"}},
		{"Audit", "Payment history (stub)", []string{"audit"}},
		{"Policy", "Active policy (stub)", []string{"policy"}},
		{"Balance", "Wallet balance via daemon (stub)", []string{"balance"}},
		{"Approve", "Approve pending payment (stub)", []string{"approve"}},
		{"Version", "Print CLI version", []string{"version"}},
		{"Quit", "Close this menu", nil},
	}

	m0 := tuiMenuModel{choices: choices}
	p := tea.NewProgram(m0, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	picked, ok := final.(tuiPickedModel)
	if !ok {
		return nil
	}
	if picked.argv == nil {
		return nil
	}

	c := exec.Command(os.Args[0], picked.argv...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}
