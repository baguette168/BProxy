package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bproxy/bproxy/admin"
	"github.com/bproxy/bproxy/pkg/topology"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

	activeNodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	deadNodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Strikethrough(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Background(lipgloss.Color("#333333")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Padding(1, 2)

	consoleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Padding(1, 2)
)

type tickMsg time.Time

type Model struct {
	admin         *admin.Admin
	selectedIndex int
	nodes         []*topology.NodeInfo
	consoleInput  string
	consoleOutput []string
	width         int
	height        int
}

func NewModel(adminServer *admin.Admin) Model {
	return Model{
		admin:         adminServer,
		selectedIndex: 0,
		consoleOutput: []string{"BProxy Admin Console - Ready"},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}

		case "down", "j":
			if m.selectedIndex < len(m.nodes)-1 {
				m.selectedIndex++
			}

		case "enter":
			if m.selectedIndex < len(m.nodes) {
				node := m.nodes[m.selectedIndex]
				m.consoleOutput = append(m.consoleOutput,
					fmt.Sprintf("Selected: %s (%s)", node.ID, node.Hostname))
				if len(m.consoleOutput) > 10 {
					m.consoleOutput = m.consoleOutput[1:]
				}
			}

		case "r":
			m.consoleOutput = append(m.consoleOutput, "Refreshing topology...")
			if len(m.consoleOutput) > 10 {
				m.consoleOutput = m.consoleOutput[1:]
			}

		case "h":
			m.consoleOutput = []string{
				"Keyboard Shortcuts:",
				"↑/k: Move up",
				"↓/j: Move down",
				"Enter: Select node",
				"r: Refresh",
				"h: Help",
				"q/Ctrl+C: Quit",
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.nodes = m.admin.GetTopology().GetAllNodes()
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := titleStyle.Render("🔥 BProxy - Red Team Proxy Tool 🔥")

	topologyView := m.renderTopology()
	consoleView := m.renderConsole()

	topologyBox := boxStyle.Width(m.width/2 - 4).Render(topologyView)
	consoleBox := consoleStyle.Width(m.width/2 - 4).Render(consoleView)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, topologyBox, consoleBox)

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Press 'h' for help | 'q' to quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		mainContent,
		"",
		footer,
	)
}

func (m Model) renderTopology() string {
	if len(m.nodes) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("No agents connected")
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Render("📡 Agent Topology"))
	sb.WriteString("\n\n")

	for i, node := range m.nodes {
		prefix := "  "
		if i == m.selectedIndex {
			prefix = "▶ "
		}

		status := "●"
		style := activeNodeStyle
		if !node.IsActive {
			status = "○"
			style = deadNodeStyle
		}

		if i == m.selectedIndex {
			style = selectedStyle
		}

		nodeInfo := fmt.Sprintf("%s %s %s", status, node.ID[:8], node.Hostname)
		if len(node.LocalIPs) > 0 {
			nodeInfo += fmt.Sprintf(" [%s]", node.LocalIPs[0])
		}

		line := prefix + style.Render(nodeInfo)
		sb.WriteString(line)
		sb.WriteString("\n")

		if node.ParentID != "" {
			sb.WriteString(fmt.Sprintf("    ↳ Parent: %s\n", node.ParentID[:8]))
		}

		if len(node.Children) > 0 {
			sb.WriteString(fmt.Sprintf("    ↳ Children: %d\n", len(node.Children)))
		}

		lastSeen := time.Since(node.LastSeen)
		sb.WriteString(fmt.Sprintf("    ↳ Last seen: %s ago\n", lastSeen.Round(time.Second)))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m Model) renderConsole() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF00FF")).
		Render("💻 Console"))
	sb.WriteString("\n\n")

	agents := m.admin.GetAgents()
	sb.WriteString(fmt.Sprintf("Active Connections: %d\n\n", len(agents)))

	sb.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render("Recent Activity:"))
	sb.WriteString("\n")

	for _, line := range m.consoleOutput {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC")).
			Render("  " + line))
		sb.WriteString("\n")
	}

	return sb.String()
}

func RunTUI(adminServer *admin.Admin) error {
	p := tea.NewProgram(NewModel(adminServer), tea.WithAltScreen())
	_, err := p.Run()
	return err
}