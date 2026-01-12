package main

import (
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	robotdiff "robot_diff/backend/diff"
	backend "robot_diff/backend/server"
)

//go:embed template/report.html
var htmlTemplate string

//go:embed template/styles.css
var cssTemplate string

//go:embed template/app.js
var jsTemplate string

const usage = `Diff Tool for Robot Framework Outputs

Usage:  robotdiff [options] input_files

This script compares two or more Robot Framework output files and creates a
report where possible differences between test case statuses in each file
are highlighted. Main use case is verifying that results from executing same
test cases in different environments are same.

Options:
 -r --report file         HTML report file (created from the input files).
                          Default is 'robotdiff.html'.
 -n --name name           Custom names for test runs. If this option is used,
                          it must be used as many times as there are input
                          files. By default test run names are got from the
                          input file names.
 -t --title title         Title for the generated diff report. The default
                          title is 'Test Run Diff Report'.
 --enable-history         Enable history saving feature in the report. This
                          allows you to save comparison results with tags and
                          view historical trends.
 --history-file file      Path to history storage file. Default is
                          'robotdiff_history.json'.
 -h --help                Print this usage instruction.

Examples:
$ robotdiff output1.xml output2.xml output3.xml
$ robotdiff --name Env1 --name Env2 smoke1.xml smoke2.xml
$ robotdiff --enable-history --name Run1 --name Run2 output1.xml output2.xml
`

type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Config struct {
	Report        string
	Names         StringSlice
	Title         string
	Help          bool
	HistoryFile   string
	ViewHistory   string
	HistoryEnable bool
	Serve         bool
	Dir           string
	Addr          string
	ScanInterval  time.Duration
}

func main() {
	config := parseArgs()

	if config.Help {
		fmt.Print(usage)
		os.Exit(0)
	}

	args := flag.Args()
	if shouldServe(config, args) {
		dir := config.Dir
		if dir == "" && len(args) == 1 {
			dir = args[0]
		}
		if dir == "" {
			fmt.Fprintln(os.Stderr, "Error: missing directory")
			os.Exit(1)
		}
		if config.Addr == "" {
			config.Addr = ":8080"
		}
		if config.ScanInterval <= 0 {
			config.ScanInterval = 2 * time.Second
		}

		store := backend.NewRunStore(dir, config.ScanInterval)
		store.Start()
		server := backend.NewServer(config.Addr, store)
		fmt.Printf("Serving on http://localhost%s (watching %s)\n", normalizeLocalhostAddr(config.Addr), dir)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	inputFiles := args
	if len(inputFiles) < 2 {
		fmt.Fprintln(os.Stderr, "Error: At least 2 input files are required")
		fmt.Print(usage)
		os.Exit(1)
	}

	names := getNames(config.Names, inputFiles)
	if names == nil {
		os.Exit(1)
	}

	results := robotdiff.NewDiffResults()

	// Parse files in parallel
	type parseResult struct {
		index int
		robot *robotdiff.Robot
		err   error
	}

	resultChan := make(chan parseResult, len(inputFiles))
	var wg sync.WaitGroup

	for i, path := range inputFiles {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			data, err := os.ReadFile(p)
			if err != nil {
				resultChan <- parseResult{idx, nil, fmt.Errorf("failed to read file: %w", err)}
				return
			}
			robot, err := robotdiff.ParseRobotXMLBytes(data)
			if err != nil {
				resultChan <- parseResult{idx, nil, fmt.Errorf("failed to parse XML: %w", err)}
				return
			}
			resultChan <- parseResult{idx, robot, nil}
		}(i, path)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	parsedResults := make([]*robotdiff.Robot, len(inputFiles))
	for result := range resultChan {
		if result.err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", inputFiles[result.index], result.err)
			os.Exit(1)
		}
		parsedResults[result.index] = result.robot
	}

	// Add results sequentially to maintain order
	for i, robot := range parsedResults {
		results.AddParsedOutput(robot, names[i])
	}

	reporter := robotdiff.NewDiffReporter(
		config.Report,
		config.Title,
		names,
		inputFiles,
		robotdiff.Templates{HTML: htmlTemplate, CSS: cssTemplate, JS: jsTemplate},
	)
	if err := reporter.Report(results, config.HistoryFile, config.HistoryEnable); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report: %s\n", reporter.OutPath)
}

func parseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.Report, "r", "robotdiff.html", "HTML report file")
	flag.StringVar(&config.Report, "report", "robotdiff.html", "HTML report file")
	flag.Var(&config.Names, "n", "Custom name for test run (can be used multiple times)")
	flag.Var(&config.Names, "name", "Custom name for test run (can be used multiple times)")
	flag.StringVar(&config.Title, "t", "Test Run Diff Report", "Title for the diff report")
	flag.StringVar(&config.Title, "title", "Test Run Diff Report", "Title for the diff report")
	flag.BoolVar(&config.Help, "h", false, "Show help")
	flag.BoolVar(&config.Help, "help", false, "Show help")
	flag.StringVar(&config.HistoryFile, "history-file", "robotdiff_history.json", "Path to history storage file")
	flag.StringVar(&config.ViewHistory, "view-history", "", "View history for a specific tag")
	flag.BoolVar(&config.HistoryEnable, "enable-history", false, "Enable history saving feature in the report")
	flag.BoolVar(&config.Serve, "serve", false, "Run HTTP server (expects --dir or a directory arg)")
	flag.StringVar(&config.Dir, "dir", "", "Directory to scan for Robot XML outputs")
	flag.StringVar(&config.Addr, "addr", ":8080", "HTTP listen address")
	flag.DurationVar(&config.ScanInterval, "scan-interval", 2*time.Second, "Directory scan interval")

	flag.Usage = func() {
		fmt.Print(usage)
	}

	flag.Parse()
	return config
}

func shouldServe(config *Config, args []string) bool {
	if config.Serve {
		return true
	}
	if config.Dir != "" {
		return true
	}
	if len(args) == 1 {
		fi, err := os.Stat(args[0])
		return err == nil && fi.IsDir()
	}
	return false
}

func normalizeLocalhostAddr(addr string) string {
	// Print a friendly localhost URL for default cases.
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	return addr
}

func getNames(names []string, paths []string) []string {
	if len(names) == 0 {
		return paths
	}
	if len(names) == len(paths) {
		return names
	}
	fmt.Fprintf(os.Stderr, "Different number of test run names (%d) and input files (%d).\n",
		len(names), len(paths))
	return nil
}
