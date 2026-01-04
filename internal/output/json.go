package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
)

// JSONOutput represents the JSON structure
type JSONOutput struct {
	TotalGoroutines   int                            `json:"total_goroutines"`
	PeakGoroutines    int                            `json:"peak_goroutines"`
	TotalBlockedTime  string                         `json:"total_blocked_time"`
	TotalRuntime      string                         `json:"total_runtime"`
	BlockingBreakdown map[string]BlockingReasonStats `json:"blocking_breakdown"`
	TopBlocked        []GoroutineJSON                `json:"top_blocked_goroutines"`
	PerformanceIssues bool                           `json:"has_performance_issues"`
	Issues            []string                       `json:"issues,omitempty"`
}

// BlockingReasonStats contains stats for a blocking reason
type BlockingReasonStats struct {
	Duration   string  `json:"duration"`
	Percentage float64 `json:"percentage"`
}

// GoroutineJSON represents a goroutine in JSON
type GoroutineJSON struct {
	ID               uint64            `json:"id"`
	TotalBlocked     string            `json:"total_blocked"`
	TotalRuntime     string            `json:"total_runtime"`
	TotalRunnable    string            `json:"total_runnable"`
	PrimaryReason    string            `json:"primary_blocking_reason"`
	BlockingEvents   int               `json:"blocking_events_count"`
	BlockingByReason map[string]string `json:"blocking_by_reason,omitempty"`
}

// JSONFormatter handles JSON output
type JSONFormatter struct {
	writer io.Writer
}

// NewJSONFormatter creates a JSON formatter
func NewJSONFormatter(w io.Writer) *JSONFormatter {
	return &JSONFormatter{writer: w}
}

// FormatSummary outputs the summary as JSON
func (f *JSONFormatter) FormatSummary(summary *model.Summary) error {
	output := f.convertToJSON(summary)

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")

	return encoder.Encode(output)
}

// FormatGoroutineDetail outputs goroutine details as JSON
func (f *JSONFormatter) FormatGoroutineDetail(g *model.GoroutineInfo) error {
	output := f.convertGoroutineToJSON(g, true)

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", "  ")

	return encoder.Encode(output)
}

// convertToJSON transforms model.Summary to JSONOutput
func (f *JSONFormatter) convertToJSON(summary *model.Summary) *JSONOutput {
	output := &JSONOutput{
		TotalGoroutines:   summary.TotalGoroutines,
		PeakGoroutines:    summary.PeakGoroutines,
		TotalBlockedTime:  formatDurationJSON(summary.TotalBlockedTime),
		TotalRuntime:      formatDurationJSON(summary.TotalRuntime),
		BlockingBreakdown: make(map[string]BlockingReasonStats),
		TopBlocked:        make([]GoroutineJSON, 0, len(summary.TopBlocked)),
		PerformanceIssues: summary.HasPerformanceIssues,
		Issues:            summary.Issues,
	}

	for reason, duration := range summary.BlockingBreakdown {
		output.BlockingBreakdown[reason.String()] = BlockingReasonStats{
			Duration:   formatDurationJSON(duration),
			Percentage: summary.BlockingPercent[reason],
		}
	}

	for _, g := range summary.TopBlocked {
		output.TopBlocked = append(output.TopBlocked, f.convertGoroutineToJSON(g, false))
	}

	return output
}

// convertGoroutineToJSON transforms model.GoroutineInfo to GoroutineJSON
func (f *JSONFormatter) convertGoroutineToJSON(g *model.GoroutineInfo, includeDetails bool) GoroutineJSON {
	gj := GoroutineJSON{
		ID:             g.ID,
		TotalBlocked:   formatDurationJSON(g.TotalBlocked),
		TotalRuntime:   formatDurationJSON(g.TotalRuntime),
		TotalRunnable:  formatDurationJSON(g.TotalRunnable),
		PrimaryReason:  getPrimaryReason(g).String(),
		BlockingEvents: len(g.BlockingEvents),
	}

	if includeDetails {
		gj.BlockingByReason = make(map[string]string)
		for reason, duration := range g.BlockingByReason {
			if duration > 0 {
				gj.BlockingByReason[reason.String()] = formatDurationJSON(duration)
			}
		}
	}

	return gj
}

// formatDurationJSON converts duration to string for JSON
func formatDurationJSON(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Microsecond {
		return d.String()
	}
	if d < time.Millisecond {
		return d.String()
	}
	if d < time.Second {
		return d.String()
	}

	return d.String()
}

// getPrimaryReason finds the dominant blocking reason
func getPrimaryReason(g *model.GoroutineInfo) model.BlockingReason {
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
