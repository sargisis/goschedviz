# goschedviz

A production-quality Go CLI tool that analyzes runtime trace files and explains scheduler behavior and performance bottlenecks.

## Overview

`goschedviz` parses Go execution traces (generated via `go test -trace` or `runtime/trace`) and provides human-readable insights into:

- **Goroutine lifecycle** and state transitions
- **Blocking patterns** (channel operations, mutex contention, syscalls, GC pauses)
- **Scheduler behavior** (running/runnable/blocked states)
- **Performance bottlenecks** with actionable explanations

Unlike raw trace viewers, `goschedviz` **explains WHY your program is slow**, not just what happened.

## Installation

```bash
go install github.com/goschedviz/goschedviz/cmd/goschedviz@latest
```

Or build from source:

```bash
git clone https://github.com/goschedviz/goschedviz
cd goschedviz
go build -o goschedviz ./cmd/goschedviz
```

## 🌍 Real-World Workflow

**1. Capture a Trace**
Run your Go application and capture a trace file using the standard library or curl:
```bash
# Option A: From code (runtime/trace)
f, _ := os.Create("trace.out")
trace.Start(f)
defer trace.Stop()

# Option B: From a running server (net/http/pprof)
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
```

**2. Analyze & Pinpoint**
Check for high-level issues. If the command exits with code 2, performance alerts were triggered.
```bash
goschedviz analyze trace.out
```

**3. Deep Dive**
Use the interactive explorer or insights engine to find the root cause.
```bash
goschedviz explore trace.out
# or
goschedviz insights trace.out
```

## Usage Examples

### Basic Analysis

```bash
$ goschedviz trace.out

GOSCHEDVIZ TRACE ANALYSIS
================================================================================

SUMMARY
--------------------------------------------------------------------------------
Total goroutines:     1423
Peak goroutines:      3821
Total blocked time:   12.5s
Total runtime:        4.2s

BLOCKING BREAKDOWN
--------------------------------------------------------------------------------
channel receive:     48.2%  (6.02s)
mutex lock:          27.1%  (3.39s)
syscall:             15.3%  (1.91s)
GC:                  9.4%   (1.18s)

TOP BLOCKED GOROUTINES
--------------------------------------------------------------------------------
Goroutine ID    Blocked Time Primary Reason
--------------------------------------------------------------------------------
#1234           2.10s        channel receive
#987            1.40s        mutex lock
#2456           890.00ms     syscall
#3142           720.00ms     channel receive
#1891           650.00ms     mutex lock

⚠️  PERFORMANCE ISSUES DETECTED
--------------------------------------------------------------------------------
1. Excessive channel receive blocking (>40%)
2. High mutex contention (>30%)
```

### JSON Output

```bash
$ goschedviz --json trace.out

{
  "total_goroutines": 1423,
  "peak_goroutines": 3821,
  "total_blocked_time": "12.5s",
  "total_runtime": "4.2s",
  "blocking_breakdown": {
    "channel receive": {
      "duration": "6.02s",
      "percentage": 48.2
    },
    "mutex lock": {
      "duration": "3.39s",
      "percentage": 27.1
    }
  },
  "top_blocked_goroutines": [
    {
      "id": 1234,
      "total_blocked": "2.10s",
      "total_runtime": "320ms",
      "total_runnable": "180ms",
      "primary_blocking_reason": "channel receive",
      "blocking_events_count": 342
    }
  ],
  "has_performance_issues": true,
  "issues": [
    "Excessive channel receive blocking (>40%)",
    "High mutex contention (>30%)"
  ]
}
```

### Exit Codes

- **0**: Normal execution, no performance issues
- **1**: Error (invalid trace file, parsing failure)
- **2**: Performance issues detected (use in CI to fail builds with scheduler problems)

```bash
goschedviz trace.out
if [ $? -eq 2 ]; then
  echo "Performance bottlenecks detected!"
fi
```

## Architecture

### Package Structure

```
goschedviz/
├── cmd/goschedviz/         # CLI entry point (thin delegation layer)
│   └── main.go
├── internal/
│   ├── model/              # Core data types (GoroutineInfo, BlockingEvent, Summary)
│   │   └── types.go
│   ├── traceparser/        # Concurrent trace file parser
│   │   └── parser.go
│   ├── scheduler/          # Goroutine state machine reconstruction
│   │   └── statemachine.go
│   ├── analyzer/           # Performance bottleneck detection
│   │   └── analyzer.go
│   ├── stats/              # Metrics aggregation
│   │   └── aggregator.go
│   └── output/             # Human-readable & JSON formatters
│       ├── formatter.go
│       └── json.go
└── README.md
```
### Usage

1. **Generate a trace** from your application:
   ```go
   f, _ := os.Create("trace.out")
   trace.Start(f)
   defer trace.Stop()
   ```

2. **Analyze** with `goschedviz`:
   ```bash
   # Standard summary
   ./goschedviz trace.out

   # Detailed goroutine drill-down
   ./goschedviz --goroutine 42 trace.out
   ```

---

## 🛠️ Commands

| Flag | Description |
| :--- | :--- |
| `--top-blocked` | Only show the top N blocked goroutines. |
| `--goroutine <ID>` | Show deep-dive timeline and metrics for a specific ID. |
| `--json` | Output analysis results in structured JSON format. |

---

## ⚖️ Performance Indicators

⚡️ Goschedviz

> **Interactive Visualization for the Go Scheduler & Execution Tracer**

`goschedviz` is a TUI (Terminal User Interface) tool that helps you analyze why your Go programs are waiting. It parses Go execution traces to visualize **goroutine latency**, **blocking events**, and **bottlenecks** in real-time or from file.

---

## ✨ Features

*   **🖥 Unified Dashboard**: New TUI 3.0 "Command Center" (Single entry point).
*   **📡 Live Monitor**: Connect to any running Go app via Pprof and watch metrics in real-time.
*   **📊 Bottleneck Analysis**: Instantly see `Blocked` vs `Runtime` ratios.
*   **🔍 Interactive Explorer**: Sort, Filter, and Inspect individual goroutines with vim-like navigation.
*   **🔌 Framework Support**: Works with standard `net/http`, Gin, Echo, and Chi.

---

## 🚀 Installation

### Option 1: Go Install (Recommended)
```bash
go install github.com/goschedviz/goschedviz/cmd/goschedviz@latest
```
Now you can run it effectively from anywhere:
```bash
goschedviz
```

### Option 2: Docker
Build and run with creating a container:
```bash
docker build -t goschedviz .
docker run -it --net=host goschedviz
```
*(Note: `--net=host` is required to access localhost Pprof endpoints)*

---

## 🎮 How to Use

### 1. Launch the Dashboard
Simply run the command without arguments:
```bash
goschedviz
```
You will see the main menu:
1.  **Connect to Live App**: Enter your server URL (default: `localhost:6060`).
2.  **Analyze Local File**: Open an existing `trace.out` file.

### 2. Enable Pprof in Your App
Your application must expose pprof endpoints.

#### Standard Go (net/http)
```go
import _ "net/http/pprof"

func main() {
    go http.ListenAndServe("localhost:6060", nil)
    // ... your app logic
}
```

#### Gin Framework
```go
import "github.com/gin-contrib/pprof"

func main() {
    r := gin.Default()
    pprof.Register(r) // Registers /debug/pprof/*
    r.Run(":8080")
}
```

#### Echo Framework
```go
e.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))
```

---

## ⌨️ Controls

| Key | Action |
| :--- | :--- |
| `↑` / `↓` | Navigate menu / list |
| `Enter` | Select / Inspect details |
| `s` | **Sort** (Blocked / Runtime / ID) |
| `f` | **Filter** (Channels, Mutex, Network...) |
| `q` / `Esc` | Quit / Back |

---

## 🛠 Troubleshooting

### "404 Not Found" / "Connection Refused"
*   **Gin/Chi/Echo**: See the Framework examples above. Standard import doesn't work out-of-the-box.
*   **Port**: Make sure the port in `http://localhost:<PORT>/...` matches your server.

### "Invalid trace data"
*   Ensure the URL ends with `/debug/pprof/trace`.
*   Don't use `/debug/pprof/profile` (that's CPU profile, not Trace).

---

## 🏗 Architecture

The project follows a clean internal package structure:

- `internal/traceparser`: Concurrent sharded event processing.
- `internal/scheduler`: State-machine logic and transition tracking.
- `internal/analyzer`: Heuristic-based bottleneck detection.
- `internal/output`: Lip Gloss and JSON formatting layers.

---

**Note**: This tool requires Go 1.21+ and supports the latest experimental trace formats (including Go 1.25+).

### ⚠️ Trace Version Compatibility

If you encounter "unknown or unsupported trace version" errors:
```bash
go get -u golang.org/x/exp/trace
go mod tidy
```