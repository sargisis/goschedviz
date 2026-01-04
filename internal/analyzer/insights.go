package analyzer

import (
	"fmt"
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
)

// NarrativeInsight represents a high-level human-readable observation
type NarrativeInsight struct {
	Title       string
	Observation string
	Suggestion  string
	Severity    string // info, warning, critical
}

// GenerateInsights analyzes a summary and creates human-like narratives
func GenerateInsights(summary *model.Summary) []NarrativeInsight {
	var insights []NarrativeInsight

	// 1. Channel Blocking Analysis
	if summary.BlockingPercent[model.BlockChannelRecv] > 40 {
		insights = append(insights, NarrativeInsight{
			Title:       "Channel Bottleneck Detected",
			Observation: fmt.Sprintf("Your application is spending %.1f%% of its total blocked time waiting for channel receives.", summary.BlockingPercent[model.BlockChannelRecv]),
			Suggestion:  "This often indicates 'Slow Producers' or unbuffered channels causing synchronization stalls. Consider increasing channel buffers or balancing workload.",
			Severity:    "critical",
		})
	}

	// 2. Starvation Analysis
	if summary.HasPerformanceIssues {
		for _, issue := range summary.Issues {
			if issue == "Goroutine starvation detected (long runnable but not scheduled)" {
				insights = append(insights, NarrativeInsight{
					Title:       "CPU Starvation",
					Observation: "I noticed several goroutines are ready to run (Runnable) but are waiting too long for a CPU slot.",
					Suggestion:  "This usually happens when GOMAXPROCS is too low or when a few goroutines are 'hogging' the CPU with tight loops. Check for non-preemptive code.",
					Severity:    "warning",
				})
			}
		}
	}

	// 3. GC Pressure
	if summary.BlockingPercent[model.BlockGC] > 15 {
		insights = append(insights, NarrativeInsight{
			Title:       "High GC Pressure",
			Observation: fmt.Sprintf("Garbage Collection is responsible for %.1f%% of system pauses.", summary.BlockingPercent[model.BlockGC]),
			Suggestion:  "High GC overhead often stems from excessive short-lived allocations. Try using sync.Pool to reuse objects and profile memory with 'go tool pprof --alloc_objects'.",
			Severity:    "warning",
		})
	}

	// 4. General Positive Insight
	if !summary.HasPerformanceIssues && summary.TotalGoroutines > 0 {
		insights = append(insights, NarrativeInsight{
			Title:       "Healthy Scheduler State",
			Observation: "The scheduler seems well-balanced. No significant contention or starvation was detected.",
			Suggestion:  "Continue monitoring as you scale. Your current synchronization strategy is performing efficiently.",
			Severity:    "info",
		})
	}

	return insights
}

// formatDuration converts duration to human-readable string (helper)
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
