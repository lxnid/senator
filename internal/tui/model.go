package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lxnid/senator/internal/council"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1) // Purple background for active tab

	securityColor = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")) // Red/Pink
	logicColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7AA")) // Teal
	styleColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A7FF")) // Blue
)

type ValidationState int

const (
	StateLoading ValidationState = iota
	StateReviewing
	StateDone
)

type Model struct {
	Code      string
	Context   string
	Tabs      []string
	ActiveTab int
	Spinner   spinner.Model
	Viewport  viewport.Model
	State     ValidationState
	Reviews   map[string]string // role -> content
	Summary   string
	Err       error
	Ready     bool
	
	// Channels for streaming updates could go here
}

func NewModel(code, contextInfo string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		Code:      code,
		Context:   contextInfo,
		Tabs:      []string{"Code", "Security", "Logic", "Style", "Verdict"},
		ActiveTab: 0,
		Spinner:   s,
		State:     StateLoading,
		Reviews:   make(map[string]string),
	}
}

func (m Model) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right":
			m.ActiveTab = (m.ActiveTab + 1) % len(m.Tabs)
			m.Viewport.SetContent(m.renderContent())
			m.Viewport.GotoTop()
		case "shift+tab", "left":
			m.ActiveTab = (m.ActiveTab - 1 + len(m.Tabs)) % len(m.Tabs)
			m.Viewport.SetContent(m.renderContent())
			m.Viewport.GotoTop()
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		
		if !m.Ready {
			m.Viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.Viewport.HighPerformanceRendering = false
			m.Viewport.SetContent(m.renderContent())
			m.Ready = true
		} else {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - verticalMarginHeight
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		// Only update content if in loading state and on a relevant tab to show spinner animation
		if m.State == StateLoading && m.renderContent() != m.Viewport.View() {
			m.Viewport.SetContent(m.renderContent())
		}
		return m, cmd

	case council.SenateReport:
		m.State = StateDone
		m.Summary = msg.Summary
		m.ActiveTab = 4 // Switch to Verdict
		m.Viewport.SetContent(m.renderContent())
		
	case council.ReviewResult:
		content := msg.RawOutput
		if msg.Warning != "" {
			content = "⚠️ " + msg.Warning + "\n\n" + content
		}
		if msg.Error != nil {
			content += fmt.Sprintf("\n\n⚠️ Error: %v", msg.Error)
		}
		m.Reviews[msg.ReviewerType] = content
		// If currently viewing this tab, update viewport
		currentTab := m.Tabs[m.ActiveTab]
		if strings.EqualFold(currentTab, msg.ReviewerType) {
			m.Viewport.SetContent(m.renderContent())
		}
	}

	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.Ready {
		return "\n  Initializing Senate TUI...\n"
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.Viewport.View(), m.footerView())
}

func (m Model) headerView() string {
	doc := strings.Builder{}
	// Title
	doc.WriteString(titleStyle.Render("🏛️  The Senate"))
	doc.WriteString("  ")
	doc.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(m.Context))
	doc.WriteString("\n\n")

	// Tabs
	var tabs []string
	for i, t := range m.Tabs {
		if i == m.ActiveTab {
			tabs = append(tabs, activeTabStyle.Render(t))
		} else {
			tabs = append(tabs, tabStyle.Render(t))
		}
	}
	doc.WriteString(strings.Join(tabs, " "))
	return doc.String()
}

func (m Model) footerView() string {
	return "\n(tab: switch view • q: quit)"
}

func (m Model) renderContent() string {
	content := ""
	switch m.Tabs[m.ActiveTab] {
	case "Code":
		content = m.Code
	case "Security":
		content = m.Reviews["security"]
	case "Logic":
		content = m.Reviews["logic"]
	case "Style":
		content = m.Reviews["style"]
	case "Verdict":
		if m.State == StateLoading {
			content = fmt.Sprintf("%s The Senate is deliberating...", m.Spinner.View())
		} else if m.Summary != "" {
			content = m.Summary
		} else {
			content = "Waiting for synthesis..."
		}
	}

	if content == "" {
		content = fmt.Sprintf("%s Waiting for testimony...", m.Spinner.View())
	}
	return content
}
