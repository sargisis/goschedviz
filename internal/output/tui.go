package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goschedviz/goschedviz/internal/model"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	detailStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Width(60)

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
)

type modelState int

const (
	stateTable modelState = iota
	stateDetail
)

type sortField int

const (
	sortBlocked sortField = iota
	sortRuntime
	sortID
)

// ExplorerModel is the bubbletea model for the interactive trace explorer
type ExplorerModel struct {
	table        table.Model
	summary      *model.Summary
	goroutines   map[uint64]*model.GoroutineInfo
	state        modelState
	selectedID   uint64
	sortField    sortField
	filterReason model.BlockingReason
}

func NewExplorerModel(summary *model.Summary, goroutines map[uint64]*model.GoroutineInfo) ExplorerModel {
	m := ExplorerModel{
		summary:      summary,
		goroutines:   goroutines,
		state:        stateTable,
		sortField:    sortBlocked,
		filterReason: model.BlockNone,
	}

	// Setup initial table
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true)
	t.SetStyles(s)

	m.table = t
	m.RefreshTable() // Populate initial data
	return m
}

func (m ExplorerModel) Init() tea.Cmd { return nil }

func (m ExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.state == stateDetail {
				m.state = stateTable
				return m, nil
			}
			// In dashboard mode, we might want to let the parent handle Quit or Back
			return m, nil
		case "s":
			m.sortField = (m.sortField + 1) % 3
			m.RefreshTable()
		case "f":
			m.cycleFilter()
			m.RefreshTable()
		case "enter":
			if m.state == stateTable {
				row := m.table.SelectedRow()
				if row == nil {
					return m, nil
				}
				idStr := row[0]
				var id uint64
				fmt.Sscanf(idStr, "#%d", &id)
				m.selectedID = id
				m.state = stateDetail
				return m, nil
			}
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *ExplorerModel) cycleFilter() {
	// ... (rest same, just receiver name change)
	switch m.filterReason {
	case model.BlockNone:
		m.filterReason = model.BlockChannelRecv
	case model.BlockChannelRecv:
		m.filterReason = model.BlockChannelSend
	case model.BlockChannelSend:
		m.filterReason = model.BlockMutexLock
	case model.BlockMutexLock:
		m.filterReason = model.BlockSyscall
	case model.BlockSyscall:
		m.filterReason = model.BlockGC
	default:
		m.filterReason = model.BlockNone
	}
}

// RefreshTable updates the table data based on current state
func (m *ExplorerModel) RefreshTable() {
	// ... logic needs to be moved here from original refreshTable
	// Copying the logic from the original file but adapting receiver
	var filtered []*model.GoroutineInfo
	for _, g := range m.goroutines {
		if m.filterReason != model.BlockNone {
			if getPrimaryBlockingReason(g) != m.filterReason {
				continue
			}
		}
		filtered = append(filtered, g)
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch m.sortField {
		case sortBlocked:
			return filtered[i].TotalBlocked > filtered[j].TotalBlocked
		case sortRuntime:
			return filtered[i].TotalRuntime > filtered[j].TotalRuntime
		case sortID:
			return filtered[i].ID < filtered[j].ID
		default:
			return filtered[i].ID < filtered[j].ID
		}
	})

	var rows []table.Row
	for _, g := range filtered {
		bar := ""
		if m.summary.TotalBlockedTime > 0 {
			pct := float64(g.TotalBlocked) / float64(m.summary.TotalBlockedTime) * 100
			width := int(pct / 2) // scale down
			if width > 10 {
				width = 10
			}
			if width > 0 {
				bar = " " + strings.Repeat("█", width)
			}
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("#%d", g.ID),
			formatDuration(g.TotalBlocked) + bar,
			formatDuration(g.TotalRuntime),
			getPrimaryBlockingReason(g).String(),
		})
	}

	columns := []table.Column{
		{Title: "ID " + m.sortIndicator(sortID), Width: 8},
		{Title: "Blocked " + m.sortIndicator(sortBlocked), Width: 20},
		{Title: "Runtime " + m.sortIndicator(sortRuntime), Width: 12},
		{Title: "Primary Reason", Width: 20},
	}

	m.table.SetColumns(columns)
	m.table.SetRows(rows)
}

func (m ExplorerModel) sortIndicator(field sortField) string {
	if m.sortField == field {
		return "↓"
	}
	return ""
}

func (m ExplorerModel) View() string {
	if m.state == stateDetail {
		return m.detailView()
	}

	// Remove the static header since Dashboard will likely provide it
	// keeping it simple for now or maybe just the stats part?
	// For now let's keep it self-contained but maybe removing the "GOSCHEDVIZ EXPLORER" title if embedded?
	// Leaving as is, but maybe we can make it conditional.

	s := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true).
		Render(" EXPLORER VIEW ")

	filterStr := "None"
	if m.filterReason != model.BlockNone {
		filterStr = m.filterReason.String()
	}

	stats := fmt.Sprintf("\n Goroutines: %d | Total Blocked: %s | Filter: %s\n",
		len(m.table.Rows()),
		formatDuration(m.summary.TotalBlockedTime),
		filterStr)

	return lipgloss.JoinVertical(lipgloss.Left,
		s,
		stats,
		baseStyle.Render(m.table.View()),
		helpStyle.Render(" • ↑/↓: navigate • s: sort • f: filter • enter: inspect • esc: back"),
	)
}

func (m ExplorerModel) detailView() string {
	// ... keep same implementation
	g := m.goroutines[m.selectedID]

	banner := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true).
		Render(fmt.Sprintf(" GOROUTINE #%d DETAILS ", g.ID))

	content := fmt.Sprintf(
		"State:     %s\nRuntime:   %s\nRunnable:  %s\nBlocked:   %s\n\nRecent Events:\n",
		g.CurrentState,
		formatDuration(g.TotalRuntime),
		formatDuration(g.TotalRunnable),
		formatDuration(g.TotalBlocked),
	)

	for i := 0; i < len(g.BlockingEvents) && i < 10; i++ {
		ev := g.BlockingEvents[i]
		content += fmt.Sprintf(" - %s (%s)\n", ev.Reason, formatDuration(ev.Duration))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		banner,
		"\n",
		detailStyle.Render(content),
		helpStyle.Render(" • esc: back to list"),
	)
}

// StartTUI launches the interactive dashboard (Legacy wrapper)
func StartTUI(summary *model.Summary, goroutines map[uint64]*model.GoroutineInfo) error {
	m := NewExplorerModel(summary, goroutines)
	// We need to wrap it to handle Quit properly if run standalone
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}
