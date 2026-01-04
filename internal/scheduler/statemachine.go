package scheduler

import (
	"time"

	"github.com/goschedviz/goschedviz/internal/model"
)

// StateTracker maintains goroutine state machines
type StateTracker struct {
	goroutines map[uint64]*goroutineState
}

// goroutineState tracks per-goroutine state with timestamps
type goroutineState struct {
	info               *model.GoroutineInfo
	lastTransitionTime time.Duration
	blockStartTime     time.Duration
	blockReason        model.BlockingReason
}

// NewStateTracker creates a state tracker
func NewStateTracker() *StateTracker {
	return &StateTracker{
		goroutines: make(map[uint64]*goroutineState),
	}
}

// RecordTransition processes a state transition and updates timing
func (st *StateTracker) RecordTransition(gid uint64, timestamp time.Duration, fromState, toState model.GoroutineState, reason model.BlockingReason) {
	gs, exists := st.goroutines[gid]
	if !exists {
		gs = &goroutineState{
			info:               model.NewGoroutineInfo(gid, timestamp),
			lastTransitionTime: timestamp,
		}
		st.goroutines[gid] = gs
	}

	duration := timestamp - gs.lastTransitionTime

	// Update time spent in previous state
	switch fromState {
	case model.StateRunning:
		gs.info.TotalRuntime += duration
	case model.StateRunnable:
		gs.info.TotalRunnable += duration
	case model.StateBlocked:
		// Blocking duration recorded separately when unblocking
	}

	// Handle state transitions
	if toState == model.StateBlocked {
		// Starting a blocking period
		gs.blockStartTime = timestamp
		gs.blockReason = reason
	} else if fromState == model.StateBlocked {
		// Ending a blocking period
		if gs.blockStartTime > 0 {
			blockDuration := timestamp - gs.blockStartTime
			event := model.BlockingEvent{
				StartTime: gs.blockStartTime,
				EndTime:   timestamp,
				Duration:  blockDuration,
				Reason:    gs.blockReason,
			}
			gs.info.AddBlockingEvent(event)
			gs.blockStartTime = 0
		}
	}

	gs.info.CurrentState = toState
	gs.lastTransitionTime = timestamp
}

// GetGoroutineInfo returns goroutine info for analysis
func (st *StateTracker) GetGoroutineInfo(gid uint64) *model.GoroutineInfo {
	if gs, exists := st.goroutines[gid]; exists {
		return gs.info
	}
	return nil
}

// GetAllGoroutines returns all tracked goroutines
func (st *StateTracker) GetAllGoroutines() map[uint64]*model.GoroutineInfo {
	result := make(map[uint64]*model.GoroutineInfo, len(st.goroutines))
	for gid, gs := range st.goroutines {
		result[gid] = gs.info
	}
	return result
}

// PeakGoroutineCount returns the maximum concurrent goroutines
func (st *StateTracker) PeakGoroutineCount() int {
	return len(st.goroutines)
}
