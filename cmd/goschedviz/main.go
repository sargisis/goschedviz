package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/goschedviz/goschedviz/internal/analyzer"
	"github.com/goschedviz/goschedviz/internal/model"
	"github.com/goschedviz/goschedviz/internal/output"
	"github.com/goschedviz/goschedviz/internal/traceparser"
)

func main() {
	if len(os.Args) < 2 {
		// TUI 3.0: Launch Unified Dashboard
		m := output.NewDashboardModel()
		if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error launching dashboard: %v\n", err)
			os.Exit(1)
		}
		return
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "analyze":
		handleAnalyze()
	case "insights":
		handleInsights()
	case "inspect":
		handleInspect()
	case "explore":
		handleExplore()
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printGeneralUsage()
	default:
		// Backward compatibility: if the first arg is a file, run analyze
		if _, err := os.Stat(subcommand); err == nil {
			handleAnalyzeLegacy(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", subcommand)
		printGeneralUsage()
		os.Exit(1)
	}
}

const Version = "1.5.0-Gemini"

func printVersion() {
	fmt.Printf("goschedviz %s\n", Version)
}

func printGeneralUsage() {
	title := output.GetTitleStyle().Render(" GOSCHEDVIZ — THE AI-READY SCHEDULER ANALYZER ")
	fmt.Println("\n" + title + "\n")

	fmt.Printf("Usage: goschedviz <command> [<args>]\n\n")
	fmt.Println("Commands:")
	fmt.Printf("  %-10s %s\n", "analyze", "Standard metrics & performance markers")
	fmt.Printf("  %-10s %s\n", "insights", "Narrative analysis and optimization suggestions")
	fmt.Printf("  %-10s %s\n", "inspect", "Deep-dive into a specific goroutine (--gid)")
	fmt.Printf("  %-10s %s\n", "explore", "Interactive TUI dashboard for trace exploration")
	fmt.Printf("  %-10s %s\n", "version", "Print current version")

	fmt.Printf("\nRun 'goschedviz <command> --help' for flags.\n")
}

func handleAnalyze() {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output in JSON format")
	topBlocked := fs.Bool("top", false, "Show only top blocked goroutines")
	watch := fs.Bool("watch", false, "Watch trace file for changes and re-analyze")
	fs.BoolVar(watch, "w", false, "Watch trace file for changes and re-analyze (shorthand)")
	fs.Parse(os.Args[2:])

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: goschedviz analyze [flags] <trace-file>\n")
		os.Exit(1)
	}

	traceFile := fs.Arg(0)
	action := func() bool {
		return runAnalysis(traceFile, *topBlocked, *jsonOutput)
	}

	if *watch {
		watchFile(traceFile, action)
		return
	}

	if !action() {
		fmt.Println("\n✖ Performance issues detected (exit code 2)")
		os.Exit(2)
	}
}

func handleInsights() {
	fs := flag.NewFlagSet("insights", flag.ExitOnError)
	watch := fs.Bool("watch", false, "Watch trace file for changes and re-analyze")
	fs.BoolVar(watch, "w", false, "Watch trace file for changes and re-analyze (shorthand)")
	fs.Parse(os.Args[2:])

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: goschedviz insights <trace-file>\n")
		os.Exit(1)
	}

	traceFile := fs.Arg(0)

	action := func() bool {
		summary, _, err := parseAndAnalyze(traceFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return false
		}
		insights := analyzer.GenerateInsights(summary)
		formatter := output.NewFormatter(os.Stdout)
		formatter.FormatInsights(insights)
		return true
	}

	if *watch {
		watchFile(traceFile, action)
		return
	}
	if !action() {
		os.Exit(1)
	}
}

func watchFile(path string, action func() bool) {
	lastMod := time.Time{}

	fmt.Printf("👀 Watching %s for changes... (Ctrl+C to stop)\n", path)

	for {
		stat, err := os.Stat(path)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if stat.ModTime().After(lastMod) {
			// Clear screen for a clean update
			fmt.Print("\033[H\033[2J")
			action()
			lastMod = stat.ModTime()
			fmt.Printf("\n👀 Last updated: %s. Watching for changes...\n", lastMod.Format("15:04:05"))
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func handleInspect() {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	gid := fs.Uint64("gid", 0, "Goroutine ID to inspect")
	jsonOutput := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(os.Args[2:])

	if fs.NArg() != 1 || *gid == 0 {
		fmt.Fprintf(os.Stderr, "Usage: goschedviz inspect --gid <id> <trace-file>\n")
		os.Exit(1)
	}

	_, goroutines, err := parseAndAnalyze(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	g, exists := goroutines[*gid]
	if !exists {
		fmt.Fprintf(os.Stderr, "Error: goroutine #%d not found\n", *gid)
		os.Exit(1)
	}

	var formatter interface {
		FormatGoroutineDetail(*model.GoroutineInfo) error
	}
	if *jsonOutput {
		formatter = output.NewJSONFormatter(os.Stdout)
	} else {
		formatter = output.NewFormatter(os.Stdout)
	}

	if err := formatter.FormatGoroutineDetail(g); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting detail: %v\n", err)
		os.Exit(1)
	}
}

func handleExplore() {
	fs := flag.NewFlagSet("explore", flag.ExitOnError)
	fs.Parse(os.Args[2:])

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: goschedviz explore <trace-file>\n")
		os.Exit(1)
	}

	summary, goroutines, err := parseAndAnalyze(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := output.StartTUI(summary, goroutines); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching TUI: %v\n", err)
		os.Exit(1)
	}
}

func handleAnalyzeLegacy(args []string) {
	// Support old-style: goschedviz [flags] file
	// Actually, easier to just redirect to analyze
	os.Args = append([]string{os.Args[0], "analyze"}, args...)
	handleAnalyze()
}

func parseAndAnalyze(traceFile string) (*model.Summary, map[uint64]*model.GoroutineInfo, error) {
	f, err := os.Open(traceFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open trace file: %w", err)
	}
	defer f.Close()

	parser := traceparser.NewParser()
	result, err := parser.Parse(f)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse trace: %w", err)
	}

	a := analyzer.NewAnalyzer(result.Goroutines)
	summary := a.Analyze()
	return summary, result.Goroutines, nil
}

func runAnalysis(traceFile string, topOnly bool, jsonFormat bool) bool {
	summary, _, err := parseAndAnalyze(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return false
	}

	var formatter interface {
		FormatSummary(*model.Summary) error
	}
	if jsonFormat {
		formatter = output.NewJSONFormatter(os.Stdout)
	} else {
		formatter = output.NewFormatter(os.Stdout)
	}

	if err := formatter.FormatSummary(summary); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting summary: %v\n", err)
		return false
	}

	return !summary.HasPerformanceIssues
}
