package model

import "time"

// GoroutineState represents the execution state of a goroutine
type GoroutineState int

const (
	StateRunning GoroutineState = iota
	StateRunnable
	StateBlocked
)

func (s GoroutineState) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateRunnable:
		return "runnable"
	case StateBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

// BlockingReason categorizes why a goroutine is blocked
type BlockingReason int

const (
	BlockNone BlockingReason = iota
	BlockChannelSend
	BlockChannelRecv
	BlockMutexLock
	BlockSyscall
	BlockGC
	BlockNetwork
	BlockSelect
	BlockSleep
	BlockSync
)

func (r BlockingReason) String() string {
	switch r {
	case BlockNone:
		return "none"
	case BlockChannelSend:
		return "channel send"
	case BlockChannelRecv:
		return "channel receive"
	case BlockMutexLock:
		return "mutex lock"
	case BlockSyscall:
		return "syscall"
	case BlockGC:
		return "GC"
	case BlockNetwork:
		return "network"
	case BlockSelect:
		return "select"
	case BlockSleep:
		return "sleep"
	case BlockSync:
		return "sync"
	default:
		return "unknown"
	}
}

// BlockingEvent represents a single blocking occurrence
type BlockingEvent struct {
	StartTime time.Duration
	EndTime   time.Duration
	Duration  time.Duration
	Reason    BlockingReason
	Stack     string
}

// GoroutineInfo tracks the complete lifecycle and behavior of a goroutine
type GoroutineInfo struct {
	ID             uint64
	CreatedAt      time.Duration
	TerminatedAt   time.Duration
	TotalRuntime   time.Duration
	TotalBlocked   time.Duration
	TotalRunnable  time.Duration
	BlockingEvents []BlockingEvent
	CurrentState   GoroutineState

	// Aggregated blocking by reason
	BlockingByReason map[BlockingReason]time.Duration

	// State machine tracking fields
	LastStateChange time.Duration
	PendingBlock    *BlockingEvent
}

// NewGoroutineInfo creates a new goroutine tracking structure
func NewGoroutineInfo(id uint64, createdAt time.Duration) *GoroutineInfo {
	return &GoroutineInfo{
		ID:               id,
		CreatedAt:        createdAt,
		CurrentState:     StateRunnable,
		LastStateChange:  createdAt,
		BlockingEvents:   make([]BlockingEvent, 0),
		BlockingByReason: make(map[BlockingReason]time.Duration),
	}
}

// AddBlockingEvent records a blocking event and updates aggregates
func (g *GoroutineInfo) AddBlockingEvent(event BlockingEvent) {
	g.BlockingEvents = append(g.BlockingEvents, event)
	g.TotalBlocked += event.Duration
	g.BlockingByReason[event.Reason] += event.Duration
}

// Summary holds aggregate metrics for the entire trace
type Summary struct {
	TotalGoroutines int
	PeakGoroutines  int

	// Total time metrics
	TotalBlockedTime time.Duration
	TotalRuntime     time.Duration

	// Blocking breakdown by reason
	BlockingBreakdown map[BlockingReason]time.Duration
	BlockingPercent   map[BlockingReason]float64

	// Top blocked goroutines
	TopBlocked []*GoroutineInfo

	// Performance issues detected
	HasPerformanceIssues bool
	Issues               []string
}

// StateTransition represents a change in goroutine state
type StateTransition struct {
	Timestamp   time.Duration
	GoroutineID uint64
	FromState   GoroutineState
	ToState     GoroutineState
	Reason      BlockingReason
}
