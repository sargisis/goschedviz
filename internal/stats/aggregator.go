package stats

import (
	"sort"
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
)

// Aggregator computes summary metrics
type Aggregator struct {
	goroutines map[uint64]*model.GoroutineInfo
}

// NewAggregator creates a statistics aggregator
func NewAggregator(goroutines map[uint64]*model.GoroutineInfo) *Aggregator {
	return &Aggregator{
		goroutines: goroutines,
	}
}

// ComputeSummary generates aggregate metrics
func (a *Aggregator) ComputeSummary() *model.Summary {
	summary := &model.Summary{
		TotalGoroutines:   len(a.goroutines),
		PeakGoroutines:    len(a.goroutines),
		BlockingBreakdown: make(map[model.BlockingReason]time.Duration),
		BlockingPercent:   make(map[model.BlockingReason]float64),
	}

	var totalBlocked time.Duration

	for _, g := range a.goroutines {
		summary.TotalBlockedTime += g.TotalBlocked
		summary.TotalRuntime += g.TotalRuntime
		totalBlocked += g.TotalBlocked

		for reason, duration := range g.BlockingByReason {
			summary.BlockingBreakdown[reason] += duration
		}
	}

	if totalBlocked > 0 {
		for reason, duration := range summary.BlockingBreakdown {
			percentage := float64(duration) / float64(totalBlocked) * 100
			summary.BlockingPercent[reason] = percentage
		}
	}

	summary.TopBlocked = a.getTopBlocked(10)

	return summary
}

// getTopBlocked returns top N goroutines by blocked time
func (a *Aggregator) getTopBlocked(n int) []*model.GoroutineInfo {
	type item struct {
		g       *model.GoroutineInfo
		blocked time.Duration
	}

	items := make([]item, 0, len(a.goroutines))
	for _, g := range a.goroutines {
		if g.TotalBlocked > 0 {
			items = append(items, item{g: g, blocked: g.TotalBlocked})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].blocked > items[j].blocked
	})

	if len(items) < n {
		n = len(items)
	}

	result := make([]*model.GoroutineInfo, n)
	for i := 0; i < n; i++ {
		result[i] = items[i].g
	}

	return result
}

// GetGoroutinesByReason returns goroutines sorted by time in specific blocking reason
func (a *Aggregator) GetGoroutinesByReason(reason model.BlockingReason, n int) []*model.GoroutineInfo {
	type item struct {
		g        *model.GoroutineInfo
		duration time.Duration
	}

	items := make([]item, 0)
	for _, g := range a.goroutines {
		if dur, ok := g.BlockingByReason[reason]; ok && dur > 0 {
			items = append(items, item{g: g, duration: dur})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].duration > items[j].duration
	})

	if len(items) < n {
		n = len(items)
	}

	result := make([]*model.GoroutineInfo, n)
	for i := 0; i < n; i++ {
		result[i] = items[i].g
	}

	return result
}
