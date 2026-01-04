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

                case "s":
                        if m.selectedIndex < len(m.nodes) {
                                node := m.nodes[m.selectedIndex]
                                if !node.IsActive {
                                        m.consoleOutput = append(m.consoleOutput,
                                                fmt.Sprintf("Error: Agent %s is offline", node.ID[:8]))
                                } else {
                                        err := m.admin.StartSocks5(1080, node.ID)
                                        if err != nil {
                                                m.consoleOutput = append(m.consoleOutput,
                                                        fmt.Sprintf("Error: %v", err))
                                        } else {
                                                m.consoleOutput = append(m.consoleOutput,
                                                        fmt.Sprintf("✓ SOCKS5 proxy started on :1080 -> %s", node.ID[:8]))
                                        }
                                }
                                if len(m.consoleOutput) > 10 {
                                        m.consoleOutput = m.consoleOutput[1:]
                                }
                        }

                case "x":
                        err := m.admin.StopSocks5(1080)
                        if err != nil {
                                m.consoleOutput = append(m.consoleOutput,
                                        fmt.Sprintf("Error: %v", err))
                        } else {
                                m.consoleOutput = append(m.consoleOutput,
                                        "✓ SOCKS5 proxy stopped on :1080")
                        }
                        if len(m.consoleOutput) > 10 {
                                m.consoleOutput = m.consoleOutput[1:]
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
                                "s: Start SOCKS5 proxy (:1080)",
                                "x: Stop SOCKS5 proxy",
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

        title := titleStyle.Render("🥖 BProxy 🥖")

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
                Render("📡 Agent Topology (Tree View)"))
        sb.WriteString("\n\n")

        rootNodes := make([]*topology.NodeInfo, 0)
        for _, node := range m.nodes {
                if node.ParentID == "" {
                        rootNodes = append(rootNodes, node)
                }
        }

        for _, root := range rootNodes {
                m.renderNodeTree(root, 0, &sb)
        }

        return sb.String()
}

func (m Model) renderNodeTree(node *topology.NodeInfo, depth int, sb *strings.Builder) {
        indent := strings.Repeat("  ", depth)
        
        nodeIndex := -1
        for i, n := range m.nodes {
                if n.ID == node.ID {
                        nodeIndex = i
                        break
                }
        }

        prefix := indent
        if depth > 0 {
                prefix += "└─ "
        }
        
        if nodeIndex == m.selectedIndex {
                prefix = indent + "▶ "
                if depth > 0 {
                        prefix = indent + "▶─ "
                }
        }

        status := "●"
        style := activeNodeStyle
        if !node.IsActive {
                status = "○"
                style = deadNodeStyle
        }

        if nodeIndex == m.selectedIndex {
                style = selectedStyle
        }

        nodeInfo := fmt.Sprintf("%s %s", status, node.ID[:8])
        if node.Hostname != "" {
                nodeInfo += fmt.Sprintf(" %s", node.Hostname)
        }
        if len(node.LocalIPs) > 0 {
                nodeInfo += fmt.Sprintf(" [%s]", node.LocalIPs[0])
        }

        line := prefix + style.Render(nodeInfo)
        sb.WriteString(line)
        sb.WriteString("\n")

        lastSeen := time.Since(node.LastSeen)
        sb.WriteString(fmt.Sprintf("%s   ↳ Last seen: %s ago\n", indent, lastSeen.Round(time.Second)))

        if len(node.Children) > 0 {
                sb.WriteString(fmt.Sprintf("%s   ↳ Children: %d\n", indent, len(node.Children)))
                
                for _, childID := range node.Children {
                        for _, childNode := range m.nodes {
                                if childNode.ID == childID {
                                        m.renderNodeTree(childNode, depth+1, sb)
                                        break
                                }
                        }
                }
        }

        sb.WriteString("\n")
}

func (m Model) renderConsole() string {
        var sb strings.Builder
        sb.WriteString(lipgloss.NewStyle().
                Bold(true).
                Foreground(lipgloss.Color("#FF00FF")).
                Render("💻 Console"))
        sb.WriteString("\n\n")

        agents := m.admin.GetAgents()
        sb.WriteString(fmt.Sprintf("Active Connections: %d\n", len(agents)))

        socks5Servers := m.admin.GetSocks5Servers()
        if len(socks5Servers) > 0 {
                sb.WriteString(lipgloss.NewStyle().
                        Foreground(lipgloss.Color("#00FF00")).
                        Render(fmt.Sprintf("SOCKS5 Proxies: %d", len(socks5Servers))))
                sb.WriteString("\n")
                for port, addr := range socks5Servers {
                        sb.WriteString(fmt.Sprintf("  • %s (port %d)\n", addr, port))
                }
        }

        sb.WriteString("\n")
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