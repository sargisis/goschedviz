package traceparser

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
	"golang.org/x/exp/trace"
)

// ParseResult contains the parsed trace data
type ParseResult struct {
	Goroutines map[uint64]*model.GoroutineInfo
	Errors     []error
}

// Parser handles concurrent parsing of trace files
type Parser struct {
	numWorkers int
}

// NewParser creates a new trace parser with specified worker count
func NewParser() *Parser {
	return &Parser{
		numWorkers: runtime.NumCPU(),
	}
}

// Parse reads and parses a trace file concurrently using sharding to ensure consistency
func (p *Parser) Parse(r io.Reader) (*ParseResult, error) {
	reader, err := trace.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace reader: %w", err)
	}

	result := &ParseResult{
		Goroutines: make(map[uint64]*model.GoroutineInfo),
		Errors:     make([]error, 0),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create sharded channels for workers
	shards := make([]chan trace.Event, p.numWorkers)
	for i := 0; i < p.numWorkers; i++ {
		shards[i] = make(chan trace.Event, 1000)
		wg.Add(1)
		go p.worker(shards[i], result, &mu, &wg)
	}

	// Read events and distribute to workers by Goroutine ID
	go func() {
		for i := range shards {
			defer close(shards[i])
		}
		for {
			ev, err := reader.ReadEvent()
			if err != nil {
				if err != io.EOF {
					mu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("read event error: %w", err))
					mu.Unlock()
				}
				break
			}

			// Shard events by Goroutine ID to ensure ordering per goroutine
			if ev.Kind() == trace.EventStateTransition {
				st := ev.StateTransition()
				if st.Resource.Kind == trace.ResourceGoroutine {
					gid := uint64(st.Resource.Goroutine())
					shards[gid%uint64(p.numWorkers)] <- ev
					continue
				}
			}
			// For non-goroutine events, or other kind of events, discard for now
			// unless needed for global context
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	return result, nil
}

// worker processes events from its dedicated shard
func (p *Parser) worker(events <-chan trace.Event, result *ParseResult, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	for ev := range events {
		p.processEvent(ev, result, mu)
	}
}

// processEvent handles a single trace event
func (p *Parser) processEvent(ev trace.Event, result *ParseResult, mu *sync.Mutex) {
	if ev.Kind() == trace.EventStateTransition {
		st := ev.StateTransition()
		p.handleStateTransition(st, ev.Time(), result, mu)
	}
}

// handleStateTransition processes goroutine state changes
func (p *Parser) handleStateTransition(st trace.StateTransition, timestamp trace.Time, result *ParseResult, mu *sync.Mutex) {
	resource := st.Resource
	gid := uint64(resource.Goroutine())

	mu.Lock()
	g, exists := result.Goroutines[gid]
	if !exists {
		g = model.NewGoroutineInfo(gid, time.Duration(timestamp))
		result.Goroutines[gid] = g
	}
	mu.Unlock()

	// Determine blocking reason
	reason := determineBlockingReason(st)
	// Map trace states to our model states
	_, to := st.Goroutine()
	toState := mapTraceState(to)

	ts := time.Duration(timestamp)
	duration := ts - g.LastStateChange

	// Update time spent in previous state
	switch g.CurrentState {
	case model.StateRunning:
		g.TotalRuntime += duration
	case model.StateRunnable:
		g.TotalRunnable += duration
	case model.StateBlocked:
		// If we were blocked, we complete the current pending block
		if g.PendingBlock != nil {
			event := *g.PendingBlock
			event.EndTime = ts
			event.Duration = ts - event.StartTime
			g.AddBlockingEvent(event)
			g.PendingBlock = nil
		}
	}

	// Update current state
	g.CurrentState = toState
	g.LastStateChange = ts

	// Start a new blocking record if entering blocked state
	if toState == model.StateBlocked {
		g.PendingBlock = &model.BlockingEvent{
			StartTime: ts,
			Reason:    reason,
			// Stack: st.Stack.String(), // Optimized: avoid expensive string conversions
		}
	}
}

// mapTraceState converts trace.GoState to model.GoroutineState
func mapTraceState(s trace.GoState) model.GoroutineState {
	switch s {
	case trace.GoRunning:
		return model.StateRunning
	case trace.GoRunnable:
		return model.StateRunnable
	case trace.GoWaiting:
		return model.StateBlocked
	default:
		return model.StateBlocked
	}
}

// determineBlockingReason analyzes state transition to determine blocking cause
func determineBlockingReason(st trace.StateTransition) model.BlockingReason {
	reason := st.Reason

	// Map trace reasons to our blocking reasons (more robust matching)
	r := strings.ToLower(reason)
	switch {
	case strings.Contains(r, "chan receive") || strings.Contains(r, "chan send"):
		if strings.Contains(r, "receive") {
			return model.BlockChannelRecv
		}
		return model.BlockChannelSend
	case strings.Contains(r, "mutex") || strings.Contains(r, "lock") || strings.Contains(r, "semacquire"):
		return model.BlockMutexLock
	case strings.Contains(r, "syscall"):
		return model.BlockSyscall
	case strings.Contains(r, "gc"):
		return model.BlockGC
	case strings.Contains(r, "select"):
		return model.BlockSelect
	case strings.Contains(r, "network") || strings.Contains(r, "poll"):
		return model.BlockNetwork
	case strings.Contains(r, "sleep") || strings.Contains(r, "timer"):
		return model.BlockSleep
	case strings.Contains(r, "sync") || strings.Contains(r, "cond") || strings.Contains(r, "wait"):
		return model.BlockSync
	default:
		return model.BlockNone
	}
}

// contains checks if string contains substring (simple helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
