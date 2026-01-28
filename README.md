# 📊 goschedviz

> **Production-quality Go CLI tool for trace analysis & performance optimization**

`goschedviz` parses Go execution traces and explains **WHY** your program is slow, pinpointing scheduler bottlenecks, mutex contention, and GC pauses with actionable insights.

---

## 🚀 Installation

### Option A: Download Pre-built Binary (Recommended)
You can download the latest binary for your OS from the [Releases](https://github.com/sargisis/goschedviz/releases) page.

```bash
# Example for Linux AMD64
curl -L https://github.com/sargisis/goschedviz/releases/latest/download/goschedviz-linux-amd64 -o goschedviz
chmod +x goschedviz
sudo mv goschedviz /usr/local/bin/
```

### Option B: Install via Go
```bash
go install github.com/sargisis/goschedviz/cmd/goschedviz@latest
```

### Option C: Build from Source
```bash
git clone https://github.com/sargisis/goschedviz
cd goschedviz
go build -o goschedviz ./cmd/goschedviz
```

---

## 🔎 Key Features

| Feature | Description |
| :--- | :--- |
| **TUI Dashboard** | Interactive terminal UI for deep-dive trace exploration. |
| **Insights Engine** | Automated analysis that explains bottlenecks in plain English. |
| **Live Profiling** | Connect to a running server's pprof endpoint directly. |
| **JSON Export** | Export analysis results for CI/CD or custom reporting. |

---

## 🌍 Real-World Workflow

**1. Capture a Trace**
Capture a trace file using the standard library or curl:
```bash
# Option A: From code
f, _ := os.Create("trace.out")
trace.Start(f)
defer trace.Stop()

# Option B: From a running server
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
```

**2. Analyze & Pinpoint**
Check for high-level issues. If the command exits with code 2, performance alerts were triggered.
```bash
goschedviz analyze trace.out
# or
goschedviz explore trace.out
# or
goschedviz insights trace.out
```
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
