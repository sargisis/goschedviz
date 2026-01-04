package output

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/goschedviz/goschedviz/internal/analyzer"
	"github.com/goschedviz/goschedviz/internal/model"
	"github.com/goschedviz/goschedviz/internal/traceparser"
)

// DashboardState enum
type dashboardState int

const (
	StateHome dashboardState = iota
	StateLiveInput
	StateManualFile
	StateExploring
	StateError
)

type DashboardModel struct {
	state          dashboardState
	explorer       ExplorerModel
	textInput      textinput.Model
	err            error
	selectedOption int
	liveURL        string
}

func NewDashboardModel() DashboardModel {
	ti := textinput.New()
	ti.Placeholder = "http://localhost:6060/debug/pprof/trace?seconds=5"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return DashboardModel{
		state:     StateHome,
		textInput: ti,
		liveURL:   "http://localhost:6060/debug/pprof/trace?seconds=5",
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global Quit handler (unless in input mode or explorer)
		if m.state == StateHome && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
		if (m.state == StateExploring) && msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	// Handle Analysis Result
	case AnalysisResultMsg:
		m.explorer = NewExplorerModel(msg.Summary, msg.Goroutines)
		m.state = StateExploring
		return m, nil

	case AnalysisErrorMsg:
		m.err = msg.Err
		m.state = StateError
		return m, nil
	}

	// State-specific updates
	switch m.state {
	case StateHome:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.selectedOption > 0 {
					m.selectedOption--
				}
			case "down", "j":
				if m.selectedOption < 2 {
					m.selectedOption++
				}
			case "enter":
				return m.handleMenuSelect()
			}
		}

	case StateLiveInput:
		var tiCmd tea.Cmd
		m.textInput, tiCmd = m.textInput.Update(msg)
		cmd = tiCmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "enter" {
				url := m.textInput.Value()
				if url == "" {
					url = m.textInput.Placeholder
				}
				m.liveURL = url
				// Start the capture/analysis loop
				return m, runLiveCapture(url)
			}
			if msg.String() == "esc" {
				m.state = StateHome
			}
		}

	case StateExploring:
		// Forward messages to the explorer sub-model
		var newExplorer tea.Model
		newExplorer, cmd = m.explorer.Update(msg)
		m.explorer = newExplorer.(ExplorerModel)

		// If user presses 'q' or 'esc' in explorer main view, go back to dashboard
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" && m.explorer.state == stateTable {
				m.state = StateHome
				return m, nil
			}
		}

	case StateError:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" || keyMsg.String() == "enter" {
				m.state = StateHome
				m.err = nil
			}
		}
	}

	return m, cmd
}

func (m DashboardModel) handleMenuSelect() (tea.Model, tea.Cmd) {
	switch m.selectedOption {
	case 0: // Connect Live
		m.state = StateLiveInput
		m.textInput.SetValue("http://localhost:6060/debug/pprof/trace?seconds=5")
		return m, nil
	case 1: // Analyze Local File
		// For simplicity/demo, just try to load "trace.out" or ask for a file picker later
		// Currently implementing a direct load for "trace.out" as a quick start,
		// or we could add a simple input state for file path.
		// Let's reuse the input state but for file path.
		return m, runFileAnalysis("trace.out")
	case 2: // Quit
		return m, tea.Quit
	}
	return m, nil
}

func (m DashboardModel) View() string {
	switch m.state {
	case StateHome:
		return m.homeView()
	case StateLiveInput:
		return m.inputView("Enter Pprof URL (seconds=5 recommended):")
	case StateExploring:
		return m.explorer.View()
	case StateError:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Border(lipgloss.DoubleBorder()).
			Padding(1).
			Render(fmt.Sprintf("Error: %v\n\nPress Esc to return", m.err))
	}
	return ""
}

func (m DashboardModel) homeView() string {
	// Simple Menu
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Render(`
  ____  _____  ____  _   _  _____ ____  __     _____ _____ 
 / ___|/ _ \ \/ ___|| | | || ____|  _ \ \ \   / /_ _|__  / 
| |  _| | | \___ \ | |_| ||  _| | | | | \ \ / / | |  / /   
| |_| | |_| |___) ||  _  || |___| |_| |  \ V /  | | / /_   
 \____|\___/|____/ |_| |_||_____|____/    \_/  |___/____|  
                     DASHBOARD v3.0
`)

	options := []string{
		"📡 Connect to Live App (Pprof)",
		"📂 Analyze 'trace.out' (Local)",
		"🚪 Quit",
	}

	menu := ""
	for i, opt := range options {
		cursor := " "
		style := lipgloss.NewStyle()
		if i == m.selectedOption {
			cursor = "👉"
			style = style.Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		}
		menu += fmt.Sprintf("%s %s\n\n", cursor, style.Render(opt))
	}

	return lipgloss.JoinVertical(lipgloss.Center, title, "\n", menu)
}

func (m DashboardModel) inputView(prompt string) string {
	return lipgloss.NewStyle().
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		Render(
			fmt.Sprintf("%s\n\n%s\n\n(Esc to cancel)", prompt, m.textInput.View()),
		)
}

// --- Commands & Messages ---

type AnalysisResultMsg struct {
	Summary    *model.Summary
	Goroutines map[uint64]*model.GoroutineInfo
}

type AnalysisErrorMsg struct {
	Err error
}

// runFileAnalysis runs the analysis logic in a background goroutine
func runFileAnalysis(filename string) tea.Cmd {
	return func() tea.Msg {
		// 1. Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return AnalysisErrorMsg{Err: fmt.Errorf("file %q not found", filename)}
		}

		// 2. Parse
		f, err := os.Open(filename)
		if err != nil {
			return AnalysisErrorMsg{Err: err}
		}
		defer f.Close()

		parser := traceparser.NewParser()
		result, err := parser.Parse(f)
		if err != nil {
			return AnalysisErrorMsg{Err: err}
		}

		// 3. Analyze
		a := analyzer.NewAnalyzer(result.Goroutines)
		summary := a.Analyze()

		return AnalysisResultMsg{
			Summary:    summary,
			Goroutines: result.Goroutines,
		}
	}
}

// runLiveCapture fetches pprof trace and then analyzes it
func runLiveCapture(url string) tea.Cmd {
	return func() tea.Msg {
		// Create a temp file
		// Use unique temp file to avoid race conditions
		out, err := os.CreateTemp("", "trace_live_*.out")
		if err != nil {
			return AnalysisErrorMsg{Err: err}
		}
		tmpFile := out.Name()
		// Defer cleanup of the temp file (optional, maybe we want to keep it if it fails for inspection?)
		// For now let's keep it to debug.
		// defer os.Remove(tmpFile)

		// Fetch from URL
		client := http.Client{Timeout: 15 * time.Second} // Bump timeout slightly
		resp, err := client.Get(url)
		if err != nil {
			out.Close()
			return AnalysisErrorMsg{Err: fmt.Errorf("failed to fetch pprof: %v", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			out.Close()
			return AnalysisErrorMsg{Err: fmt.Errorf("pprof returned status: %s", resp.Status)}
		}

		// Check for obvious non-trace content (e.g. HTML pages)
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") {
			out.Close()
			return AnalysisErrorMsg{
				Err: fmt.Errorf(
					"the URL returned a Web Page (text/html), but goschedviz needs a Binary Trace File.\n\n"+
						"  ❌ You entered:   %s\n"+
						"  ✅ You need:      http://.../debug/pprof/trace?seconds=5\n\n"+
						"This tool is for analyzing Go applications, not general websites.",
					url,
				),
			}
		}

		written, err := io.Copy(out, resp.Body)
		if err != nil {
			// explicitly close on error
			out.Close()
			return AnalysisErrorMsg{Err: err}
		}

		// Close the file to ensure all data is flushed to disk before reading
		out.Close()

		// Run analysis on the temp file
		res := runFileAnalysis(tmpFile)()

		// Enhance error message with debug info if format error
		if errMsg, ok := res.(AnalysisErrorMsg); ok {
			// Read first few bytes for debug
			f, _ := os.Open(tmpFile)
			header := make([]byte, 16)
			f.Read(header)
			f.Close()

			debugInfo := fmt.Sprintf("\n[Debug Info]\nURL: %s\nSize: %d bytes\nType: %s\nHeader: %x\nFile: %s",
				url, written, contentType, header, tmpFile)

			if strings.Contains(errMsg.Err.Error(), "not a Go execution trace") {
				return AnalysisErrorMsg{
					Err: fmt.Errorf("invalid trace data from %s.\n%s\n\nOriginal Error: %v", url, debugInfo, errMsg.Err),
				}
			}
			// Append debug info to any error
			return AnalysisErrorMsg{
				Err: fmt.Errorf("%v\n%s", errMsg.Err, debugInfo),
			}
		}
		return res
	}
}
