package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/goschedviz/goschedviz/internal/analyzer"
	"github.com/goschedviz/goschedviz/internal/model"
)

var (
	// LipGloss Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginTop(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1).
			MarginBottom(1)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9A9A9A")).
			Bold(true).
			MarginBottom(0)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	dangerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF3340")).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#56F4FA")).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	labelStyleGo = lipgloss.NewStyle().Foreground(lipgloss.Color("#9A9A9A")).Width(18)
	valStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
)

// Formatter handles human-readable output
type Formatter struct {
	writer io.Writer
}

// NewFormatter creates an output formatter
func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{writer: w}
}

func (f *Formatter) printBanner() {
	banner := `
  ____  _____  ____  _   _  _____ ____  __     _____ _____ 
 / ___|/ _ \ \/ ___|| | | || ____|  _ \ \ \   / /_ _|__  / 
| |  _| | | \___ \ | |_| ||  _| | | | | \ \ / / | |  / /  
| |_| | |_| |___) ||  _  || |___| |_| |  \ V /  | | / /_  
 \____|\___/|____/ |_| |_||_____|____/    \_/  |___/____| 
                                                           `
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	fmt.Fprintln(f.writer, style.Render(banner))
}

// FormatSummary outputs the complete analysis summary
func (f *Formatter) FormatSummary(summary *model.Summary) error {
	f.printBanner()
	fmt.Fprintln(f.writer, titleStyle.Render(" ANALYSIS COMPLETE "))

	f.writeSummarySection(summary)
	f.writeBlockingBreakdown(summary)
	f.writeTopBlocked(summary)

	if summary.HasPerformanceIssues {
		f.writePerformanceIssues(summary)
	}

	return nil
}

// writeSummarySection formats the summary metrics
func (f *Formatter) writeSummarySection(summary *model.Summary) {
	fmt.Fprintln(f.writer, headerStyle.Render(" SYSTEM SUMMARY "))
	content := []string{
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total Goroutines:"), valStyle.Render(fmt.Sprintf("%d", summary.TotalGoroutines))),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Peak Goroutines:"), valStyle.Render(fmt.Sprintf("%d", summary.PeakGoroutines))),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total Blocked:"), dangerStyle.Render(formatDuration(summary.TotalBlockedTime))),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total Runtime:"), successStyle.Render(formatDuration(summary.TotalRuntime))),
	}

	fmt.Fprintln(f.writer, borderStyle.Render(strings.Join(content, "\n")))
}

// writeBlockingBreakdown formats the blocking reason percentages
func (f *Formatter) writeBlockingBreakdown(summary *model.Summary) {
	fmt.Fprintln(f.writer, headerStyle.Render(" BLOCKING BY CATEGORY "))
	var rows []string

	type reasonPct struct {
		reason   model.BlockingReason
		pct      float64
		duration time.Duration
	}

	items := make([]reasonPct, 0)
	for reason, pct := range summary.BlockingPercent {
		items = append(items, reasonPct{
			reason:   reason,
			pct:      pct,
			duration: summary.BlockingBreakdown[reason],
		})
	}

	// Sort by percentage descending
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].pct > items[i].pct {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	for _, item := range items {
		pctStr := fmt.Sprintf("%6.1f%%", item.pct)
		var style lipgloss.Style
		if item.pct > 40 {
			style = dangerStyle
		} else if item.pct > 20 {
			style = infoStyle
		} else {
			style = successStyle
		}

		rows = append(rows, fmt.Sprintf("%s %s %s",
			labelStyleGo.Render(item.reason.String()+":"),
			style.Render(pctStr),
			mutedStyle.Render("("+formatDuration(item.duration)+")")))
	}

	fmt.Fprintln(f.writer, borderStyle.Render(strings.Join(rows, "\n")))
}

// writeTopBlocked formats the top blocked goroutines
func (f *Formatter) writeTopBlocked(summary *model.Summary) {
	if len(summary.TopBlocked) == 0 {
		return
	}

	fmt.Fprintln(f.writer, headerStyle.Render(" TOP BOTTLENECKS "))
	var rows []string
	rows = append(rows, subHeaderStyle.Render(fmt.Sprintf("%-12s %-12s %s", "GOROUTINE", "DURATION", "CAUSE")))

	for _, g := range summary.TopBlocked {
		primaryReason := getPrimaryBlockingReason(g)
		rows = append(rows, fmt.Sprintf("%-12s %-12s %s",
			infoStyle.Render(fmt.Sprintf("#%d", g.ID)),
			valStyle.Render(formatDuration(g.TotalBlocked)),
			mutedStyle.Render(primaryReason.String())))
	}

	fmt.Fprintln(f.writer, borderStyle.Render(strings.Join(rows, "\n")))
}

// writePerformanceIssues formats detected issues
func (f *Formatter) writePerformanceIssues(summary *model.Summary) {
	fmt.Fprintln(f.writer, headerStyle.Foreground(lipgloss.Color("#EF3340")).Render(" PERFORMANCE ALERTS "))
	var sb strings.Builder
	for i, issue := range summary.Issues {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
	}

	style := borderStyle.Copy().BorderForeground(lipgloss.Color("#EF3340"))
	fmt.Fprintln(f.writer, style.Render(strings.TrimSpace(sb.String())))
}

// FormatGoroutineDetail outputs detailed info for a specific goroutine
func (f *Formatter) FormatGoroutineDetail(g *model.GoroutineInfo) error {
	fmt.Fprintln(f.writer, titleStyle.Render(fmt.Sprintf(" GOROUTINE #%d ANALYSIS ", g.ID)))

	content := []string{
		fmt.Sprintf("%s %s", labelStyleGo.Render("Created at:"), formatDuration(g.CreatedAt)),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Current state:"), infoStyle.Render(g.CurrentState.String())),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total runtime:"), successStyle.Render(formatDuration(g.TotalRuntime))),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total runnable:"), valStyle.Render(formatDuration(g.TotalRunnable))),
		fmt.Sprintf("%s %s", labelStyleGo.Render("Total blocked:"), dangerStyle.Render(formatDuration(g.TotalBlocked))),
	}

	fmt.Fprintln(f.writer, headerStyle.Render(" METRICS "))
	fmt.Fprintln(f.writer, borderStyle.Render(strings.Join(content, "\n")))

	var rows []string
	rows = append(rows, subHeaderStyle.Render(fmt.Sprintf("%-12s %-12s %s", "INDEX", "DURATION", "TIMESTAMP")))

	displayCount := 10
	if len(g.BlockingEvents) < displayCount {
		displayCount = len(g.BlockingEvents)
	}

	for i := 0; i < displayCount; i++ {
		ev := g.BlockingEvents[i]
		rows = append(rows, fmt.Sprintf("%-12d %-12s %s %s",
			i+1,
			infoStyle.Render(ev.Reason.String()),
			valStyle.Render(formatDuration(ev.Duration)),
			mutedStyle.Render("@ "+formatDuration(ev.StartTime))))
	}

	if len(g.BlockingEvents) > displayCount {
		rows = append(rows, mutedStyle.Render(fmt.Sprintf("\n... and %d more events", len(g.BlockingEvents)-displayCount)))
	}

	fmt.Fprintln(f.writer, headerStyle.Render(" EVENTS TIMELINE "))
	fmt.Fprintln(f.writer, borderStyle.Render(strings.Join(rows, "\n")))
	return nil
}

// FormatInsights outputs narrative insights generated by the analyzer
func (f *Formatter) FormatInsights(insights []analyzer.NarrativeInsight) error {
	fmt.Fprintln(f.writer, titleStyle.Render(" SYSTEM INSIGHTS & OBSERVATIONS "))

	if len(insights) == 0 {
		fmt.Fprintln(f.writer, successStyle.Render("\n✨ No issues detected. Everything looks optimal!"))
		return nil
	}

	for _, insight := range insights {
		var icon string
		var colorStr string

		switch insight.Severity {
		case "critical":
			icon = "🔴"
			colorStr = "#EF3340"
		case "warning":
			icon = "🟡"
			colorStr = "#F4D03F"
		default:
			icon = "🔵"
			colorStr = "#56F4FA"
		}

		title := lipgloss.NewStyle().Foreground(lipgloss.Color(colorStr)).Bold(true).Render(fmt.Sprintf("%s %s", icon, insight.Title))
		content := fmt.Sprintf("%s\n\n%s %s",
			valStyle.Render(insight.Observation),
			infoStyle.Render("💡 Suggestion:"),
			mutedStyle.Render(insight.Suggestion))

		box := borderStyle.Copy().BorderForeground(lipgloss.Color(colorStr)).Render(content)

		fmt.Fprintln(f.writer, "\n"+title)
		fmt.Fprintln(f.writer, box)
	}

	return nil
}

// formatDuration converts duration to human-readable string
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}

	return fmt.Sprintf("%.2fs", d.Seconds())
}

// getPrimaryBlockingReason returns the reason with most time
func getPrimaryBlockingReason(g *model.GoroutineInfo) model.BlockingReason {
	var maxReason model.BlockingReason
	var maxDuration time.Duration

	for reason, duration := range g.BlockingByReason {
		if duration > maxDuration {
			maxDuration = duration
			maxReason = reason
		}
	}

	return maxReason
}

// GetTitleStyle returns the lipgloss style used for titles
func GetTitleStyle() lipgloss.Style {
	return titleStyle
}
