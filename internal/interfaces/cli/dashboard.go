package cli

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"kilometers.ai/cli/internal/core/event"
	"kilometers.ai/cli/internal/core/session"
)

// DashboardFlags holds command-line flags for the dashboard command
type DashboardFlags struct {
	SessionID    string
	RefreshRate  time.Duration
	MaxEvents    int
	MethodFilter []string
	MinRiskLevel string
}

// NewDashboardCommand creates the dashboard command
func NewDashboardCommand(container *CLIContainer) *cobra.Command {
	flags := &DashboardFlags{}

	cmd := &cobra.Command{
		Use:   "dashboard [session-id]",
		Short: "Real-time terminal dashboard for monitoring MCP events",
		Long: `Launch an interactive terminal dashboard to monitor MCP events in real-time.

The dashboard provides a live view of events with keyboard controls for filtering
and navigation. Similar to 'top' or 'htop' but for MCP event monitoring.

Examples:
  km dashboard                    # Connect to active session
  km dashboard abc123def456       # Connect to specific session
  km dashboard --method search    # Filter by method
  km dashboard --min-risk medium  # Filter by risk level`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get session ID from args or flags
			if len(args) > 0 {
				flags.SessionID = args[0]
			}

			return runDashboard(container, flags)
		},
	}

	// Add command-line flags
	cmd.Flags().DurationVar(&flags.RefreshRate, "refresh", 500*time.Millisecond, "Refresh rate for live updates")
	cmd.Flags().IntVar(&flags.MaxEvents, "max-events", 100, "Maximum number of events to display")
	cmd.Flags().StringSliceVar(&flags.MethodFilter, "method", []string{}, "Filter by specific methods")
	cmd.Flags().StringVar(&flags.MinRiskLevel, "min-risk", "low", "Minimum risk level to display (low, medium, high)")

	return cmd
}

// runDashboard starts the terminal dashboard
func runDashboard(container *CLIContainer, flags *DashboardFlags) error {
	// Find session to monitor
	sessionObj, err := findSessionForDashboard(container, flags.SessionID)
	if err != nil {
		return fmt.Errorf("failed to find session: %w", err)
	}

	// Create dashboard model
	model := newDashboardModel(container, sessionObj, flags)

	// Start the Bubble Tea program
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("dashboard failed: %w", err)
	}

	return nil
}

// findSessionForDashboard finds the session to monitor
func findSessionForDashboard(container *CLIContainer, sessionID string) (*session.Session, error) {
	if sessionID != "" {
		// Use specific session ID
		_, err := session.NewSessionID(sessionID)
		if err != nil {
			return nil, fmt.Errorf("invalid session ID: %w", err)
		}

		// For now, return an error asking for implementation
		// TODO: Need to access session repository from container
		return nil, fmt.Errorf("specific session lookup not yet implemented, please use active session")
	}

	// For MVP, we'll just create a mock active session
	// TODO: Find actual active session from session repository
	activeSession := session.NewSession(session.DefaultSessionConfig())
	if err := activeSession.Start(); err != nil {
		return nil, fmt.Errorf("failed to create mock session: %w", err)
	}

	return activeSession, nil
}

// dashboardModel holds the state for the Bubble Tea dashboard
type dashboardModel struct {
	container    *CLIContainer
	session      *session.Session
	flags        *DashboardFlags
	events       []EventDisplayItem
	selectedRow  int
	paused       bool
	lastUpdate   time.Time
	windowWidth  int
	windowHeight int
	err          error
}

// EventDisplayItem represents an event for display in the dashboard
type EventDisplayItem struct {
	Timestamp string
	Direction string
	Method    string
	Risk      string
	RiskColor lipgloss.Color
	Size      string
	Preview   string
}

// newDashboardModel creates a new dashboard model
func newDashboardModel(container *CLIContainer, sessionObj *session.Session, flags *DashboardFlags) dashboardModel {
	return dashboardModel{
		container:   container,
		session:     sessionObj,
		flags:       flags,
		events:      []EventDisplayItem{},
		selectedRow: 0,
		paused:      false,
		lastUpdate:  time.Now(),
	}
}

// Init implements the Bubble Tea init method
func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.loadEventsCmd(),
	)
}

// Update implements the Bubble Tea update method
func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case " ":
			m.paused = !m.paused
			return m, nil

		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
			}
			return m, nil

		case "down", "j":
			if m.selectedRow < len(m.events)-1 {
				m.selectedRow++
			}
			return m, nil

		case "r":
			// Force refresh
			return m, m.loadEventsCmd()
		}

	case tickMsg:
		if !m.paused {
			return m, tea.Batch(
				m.tickCmd(),
				m.loadEventsCmd(),
			)
		}
		return m, m.tickCmd()

	case eventsLoadedMsg:
		m.events = msg.events
		m.lastUpdate = time.Now()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// View implements the Bubble Tea view method
func (m dashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit", m.err)
	}

	// Header
	header := m.renderHeader()

	// Event table
	table := m.renderEventTable()

	// Footer with controls
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, table, footer)
}

// renderHeader renders the dashboard header
func (m dashboardModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("🚀 Kilometers Dashboard")

	sessionInfo := fmt.Sprintf("Session: %s | Events: %d | %s",
		m.session.ID().Value()[:8]+"...",
		len(m.events),
		m.session.Duration().Round(time.Second),
	)

	status := "LIVE"
	if m.paused {
		status = "PAUSED"
	}
	statusStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))
	if !m.paused {
		statusStyle = statusStyle.Foreground(lipgloss.Color("46"))
	}

	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		title,
		"  ",
		sessionInfo,
		"  ",
		statusStyle.Render(status),
	)

	line2 := fmt.Sprintf("Last Update: %s | Refresh Rate: %v",
		m.lastUpdate.Format("15:04:05"),
		m.flags.RefreshRate,
	)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("%s", lipgloss.Border{}.Top))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, divider)
}

// renderEventTable renders the main event table
func (m dashboardModel) renderEventTable() string {
	if len(m.events) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("\n  No events to display. Waiting for MCP activity...\n")
	}

	// Table header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("%-8s │ %-3s │ %-20s │ %-4s │ %-6s │ %s",
			"TIME", "DIR", "METHOD", "RISK", "SIZE", "PREVIEW"))

	// Table rows
	rows := []string{header}

	// Show latest events first (reverse order)
	startIdx := 0
	maxRows := m.windowHeight - 8 // Account for header and footer
	if len(m.events) > maxRows {
		startIdx = len(m.events) - maxRows
	}

	for i := len(m.events) - 1; i >= startIdx; i-- {
		event := m.events[i]

		// Style the row
		rowStyle := lipgloss.NewStyle()
		if len(m.events)-1-i == m.selectedRow {
			rowStyle = rowStyle.Background(lipgloss.Color("240"))
		}

		row := fmt.Sprintf("%-8s │ %-3s │ %-20s │ %-4s │ %-6s │ %s",
			event.Timestamp,
			event.Direction,
			truncateString(event.Method, 20),
			event.Risk,
			event.Size,
			truncateString(event.Preview, 40),
		)

		rows = append(rows, rowStyle.Render(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderFooter renders the control instructions footer
func (m dashboardModel) renderFooter() string {
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("%s", lipgloss.Border{}.Top))

	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Controls: [Space] Pause/Resume | [↑↓] Navigate | [r] Refresh | [q] Quit")

	return lipgloss.JoinVertical(lipgloss.Left, divider, controls)
}

// tickMsg is sent every refresh interval
type tickMsg time.Time

// tickCmd creates a tick command
func (m dashboardModel) tickCmd() tea.Cmd {
	return tea.Tick(m.flags.RefreshRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// eventsLoadedMsg is sent when events are loaded
type eventsLoadedMsg struct {
	events []EventDisplayItem
}

// errMsg is sent when an error occurs
type errMsg struct {
	err error
}

// loadEventsCmd loads events from the session
func (m dashboardModel) loadEventsCmd() tea.Cmd {
	return func() tea.Msg {
		// Temporarily use mock events for debugging
		events := m.loadMockEvents()
		return eventsLoadedMsg{events: events}

		/*
			// Load real events from EventStore
			criteria := ports.EventCriteria{
				SessionID: &m.session.ID(),
				Limit:     m.flags.MaxEvents,
			}

			// Apply method filtering if specified
			if len(m.flags.MethodFilter) > 0 {
				// For now, just use the first method filter
				// TODO: Support multiple method filters
				if len(m.flags.MethodFilter) > 0 {
					criteria.Method = &m.flags.MethodFilter[0]
				}
			}

			events, err := m.container.EventStore.Retrieve(criteria)
			if err != nil {
				return errMsg{err: fmt.Errorf("failed to load events: %w", err)}
			}

			// Convert events to display items
			displayItems := m.convertEventsToDisplayItems(events)

			return eventsLoadedMsg{events: displayItems}
		*/
	}
}

// loadMockEvents creates mock events for testing
func (m dashboardModel) loadMockEvents() []EventDisplayItem {
	// This will be replaced with real event loading
	now := time.Now()

	mockEvents := []EventDisplayItem{
		{
			Timestamp: now.Add(-10 * time.Second).Format("15:04:05"),
			Direction: "→",
			Method:    "search",
			Risk:      "🟡 M",
			RiskColor: lipgloss.Color("yellow"),
			Size:      "2.1K",
			Preview:   `{"query":"linear issues"}`,
		},
		{
			Timestamp: now.Add(-15 * time.Second).Format("15:04:05"),
			Direction: "←",
			Method:    "search_response",
			Risk:      "🟢 L",
			RiskColor: lipgloss.Color("green"),
			Size:      "15K",
			Preview:   `{"results":[{"id":1,"title":"Bug fix"}]}`,
		},
		{
			Timestamp: now.Add(-20 * time.Second).Format("15:04:05"),
			Direction: "→",
			Method:    "get_issue",
			Risk:      "🔴 H",
			RiskColor: lipgloss.Color("red"),
			Size:      "342B",
			Preview:   `{"issue_id":"PROJ-123"}`,
		},
	}

	return mockEvents
}

// convertEventsToDisplayItems converts core events to display items
func (m dashboardModel) convertEventsToDisplayItems(events []*event.Event) []EventDisplayItem {
	displayItems := make([]EventDisplayItem, 0, len(events))

	for _, evt := range events {
		// Determine risk color and display format
		riskDisplay := m.formatRiskDisplay(evt.RiskScore())
		riskColor := m.getRiskColor(evt.RiskScore())

		// Format direction
		direction := "→"
		if evt.Direction().String() == "inbound" {
			direction = "←"
		}

		// Format size
		size := m.formatSize(evt.Size())

		// Create preview from payload
		preview := m.createPayloadPreview(evt.Payload())

		displayItem := EventDisplayItem{
			Timestamp: evt.Timestamp().Format("15:04:05"),
			Direction: direction,
			Method:    evt.Method().Value(),
			Risk:      riskDisplay,
			RiskColor: riskColor,
			Size:      size,
			Preview:   preview,
		}

		displayItems = append(displayItems, displayItem)
	}

	return displayItems
}

// formatRiskDisplay formats the risk score for display
func (m dashboardModel) formatRiskDisplay(riskScore event.RiskScore) string {
	score := riskScore.Value()
	switch {
	case score >= 70:
		return "🔴 H"
	case score >= 30:
		return "🟡 M"
	default:
		return "🟢 L"
	}
}

// getRiskColor returns the lipgloss color for the risk level
func (m dashboardModel) getRiskColor(riskScore event.RiskScore) lipgloss.Color {
	score := riskScore.Value()
	switch {
	case score >= 70:
		return lipgloss.Color("red")
	case score >= 30:
		return lipgloss.Color("yellow")
	default:
		return lipgloss.Color("green")
	}
}

// formatSize formats the event size for display
func (m dashboardModel) formatSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fK", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1fM", float64(size)/(1024*1024))
	}
}

// createPayloadPreview creates a preview of the event payload
func (m dashboardModel) createPayloadPreview(payload []byte) string {
	preview := string(payload)

	// Remove newlines and extra whitespace
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")

	// Compress multiple spaces
	for strings.Contains(preview, "  ") {
		preview = strings.ReplaceAll(preview, "  ", " ")
	}

	return strings.TrimSpace(preview)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
