package analyzer

import (
	"sort"
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
)

// Analyzer detects performance bottlenecks and patterns
type Analyzer struct {
	goroutines map[uint64]*model.GoroutineInfo
	summary    *model.Summary
}

// NewAnalyzer creates a performance analyzer
func NewAnalyzer(goroutines map[uint64]*model.GoroutineInfo) *Analyzer {
	return &Analyzer{
		goroutines: goroutines,
		summary:    &model.Summary{},
	}
}

// Analyze performs comprehensive bottleneck detection
func (a *Analyzer) Analyze() *model.Summary {
	a.summary.TotalGoroutines = len(a.goroutines)
	a.summary.PeakGoroutines = len(a.goroutines)

	a.aggregateBlockingStats()
	a.findTopBlocked()
	a.detectPerformanceIssues()

	return a.summary
}

// aggregateBlockingStats computes blocking breakdown across all goroutines
func (a *Analyzer) aggregateBlockingStats() {
	a.summary.BlockingBreakdown = make(map[model.BlockingReason]time.Duration)
	a.summary.BlockingPercent = make(map[model.BlockingReason]float64)

	var totalBlocked time.Duration

	for _, g := range a.goroutines {
		a.summary.TotalBlockedTime += g.TotalBlocked
		a.summary.TotalRuntime += g.TotalRuntime
		totalBlocked += g.TotalBlocked

		for reason, duration := range g.BlockingByReason {
			a.summary.BlockingBreakdown[reason] += duration
		}
	}

	// Calculate percentages
	if totalBlocked > 0 {
		for reason, duration := range a.summary.BlockingBreakdown {
			percentage := float64(duration) / float64(totalBlocked) * 100
			a.summary.BlockingPercent[reason] = percentage
		}
	}
}

// findTopBlocked identifies goroutines with highest blocking time
func (a *Analyzer) findTopBlocked() {
	type blockedItem struct {
		g     *model.GoroutineInfo
		total time.Duration
	}

	items := make([]blockedItem, 0, len(a.goroutines))
	for _, g := range a.goroutines {
		if g.TotalBlocked > 0 {
			items = append(items, blockedItem{g: g, total: g.TotalBlocked})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].total > items[j].total
	})

	topN := 10
	if len(items) < topN {
		topN = len(items)
	}

	a.summary.TopBlocked = make([]*model.GoroutineInfo, topN)
	for i := 0; i < topN; i++ {
		a.summary.TopBlocked[i] = items[i].g
	}
}

// detectPerformanceIssues identifies suspicious patterns
func (a *Analyzer) detectPerformanceIssues() {
	a.summary.Issues = make([]string, 0)

	// Check for excessive channel blocking
	if pct, ok := a.summary.BlockingPercent[model.BlockChannelRecv]; ok && pct > 40 {
		a.summary.HasPerformanceIssues = true
		a.summary.Issues = append(a.summary.Issues, "Excessive channel receive blocking (>40%)")
	}

	if pct, ok := a.summary.BlockingPercent[model.BlockChannelSend]; ok && pct > 40 {
		a.summary.HasPerformanceIssues = true
		a.summary.Issues = append(a.summary.Issues, "Excessive channel send blocking (>40%)")
	}

	// Check for mutex contention
	if pct, ok := a.summary.BlockingPercent[model.BlockMutexLock]; ok && pct > 30 {
		a.summary.HasPerformanceIssues = true
		a.summary.Issues = append(a.summary.Issues, "High mutex contention (>30%)")
	}

	// Check for GC pressure
	if pct, ok := a.summary.BlockingPercent[model.BlockGC]; ok && pct > 15 {
		a.summary.HasPerformanceIssues = true
		a.summary.Issues = append(a.summary.Issues, "High GC pressure (>15%)")
	}

	// Check if single goroutine dominates blocking
	if len(a.summary.TopBlocked) > 0 {
		topBlockedPct := float64(a.summary.TopBlocked[0].TotalBlocked) / float64(a.summary.TotalBlockedTime) * 100
		if topBlockedPct > 50 {
			a.summary.HasPerformanceIssues = true
			a.summary.Issues = append(a.summary.Issues, "Single goroutine accounts for >50% of blocking time")
		}
	}

	// Check for long runnable periods (starvation detection)
	for _, g := range a.goroutines {
		if g.TotalRunnable > 0 && g.TotalRuntime > 0 {
			runnableRatio := float64(g.TotalRunnable) / float64(g.TotalRunnable+g.TotalRuntime)
			if runnableRatio > 0.7 {
				a.summary.HasPerformanceIssues = true
				a.summary.Issues = append(a.summary.Issues, "Goroutine starvation detected (long runnable but not scheduled)")
				break
			}
		}
	}
}

// GetBlockingReason returns the most common blocking reason
func (a *Analyzer) GetBlockingReason(g *model.GoroutineInfo) model.BlockingReason {
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
