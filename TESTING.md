# Testing Guide for goschedviz 🧪

This guide explains how to verify that `goschedviz` is working correctly, covering unit tests, integration tests, and manual verification.

## 1. Prerequisites

Ensure you have Go 1.25+ installed:
```bash
go version
```

## 2. Unit Tests

Run the standard Go test suite for all packages:
```bash
go test ./...
```
*(Currently, we are in the process of adding more unit coverage for `analyzer` and `traceparser`).*

## 3. Integration Testing ("The Bottleneck Demo")

We provide a specialized program `examples/bottleneck_demo.go` that simulates various performance issues (channel blocking, mutex contention, sleep).

### Step 1: Generate the Trace
```bash
cd examples
go run bottleneck_demo.go
# This will create a 'trace.out' file in the examples directory
cd ..
```

### Step 2: verify CLI Commands

#### Analyze (Standard)
```bash
go run cmd/goschedviz/main.go analyze examples/trace.out
```
**Expected Output:**
- A summary table showing ~109 goroutines.
- "Performance Alerts" section detecting "Excessive channel receive blocking".
- Exit code should be **0** (success).

#### Insights (Narrative)
```bash
go run cmd/goschedviz/main.go insights examples/trace.out
```
**Expected Output:**
- A human-readable report with "Red/Yellow" observations.
- Suggestions like "Consider increasing channel buffers".

#### Explore (Interactive TUI)
```bash
go run cmd/goschedviz/main.go explore examples/trace.out
```
**Checklist:**
- [ ] Press `s` to sort by Blocked time.
- [ ] Press `f` to filter by "Channel Recv".
- [ ] Press `Enter` on a row to see details.
- [ ] Press `q` to quit.

#### Watch Mode (Live Reload)
```bash
go run cmd/goschedviz/main.go analyze -w examples/trace.out
```
**Checklist:**
- [ ] The tool should stay running ("Watching ...").
- [ ] In another terminal, re-run `go run examples/bottleneck_demo.go`.
- [ ] Verify the tool refreshes the output automatically.

## 4. Troubleshooting
If you see "unknown command", ensure you built the binary:
```bash
go build -o goschedviz ./cmd/goschedviz
./goschedviz help
```
