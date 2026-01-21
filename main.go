package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	backend "robot_diff/backend/server"
	"robot_diff/backend/store"
)

const usage = `Robotdiff: local server + React UI for Robot Framework outputs

Usage:
	robotdiff [options] [<results-dir>]

Starts a local HTTP server and scans a directory for Robot Framework output files
(typically named 'output.xml'). If <results-dir> is omitted, the current directory
is used.

Options:
	--dir path               Directory to scan for Robot outputs (alternative to positional arg).
	--addr addr              HTTP listen address. Default: ':8080'.
	--scan-interval duration Directory scan interval. Default: 2s.
	-h, --help               Print this usage instruction.

Examples:
	robotdiff
	robotdiff .
	robotdiff --addr :3000 /path/to/results
	robotdiff --dir /path/to/results
`

type Config struct {
	Help         bool
	Dir          string
	Addr         string
	ScanInterval time.Duration
}

func main() {
	config := parseArgs()

	if config.Help {
		fmt.Print(usage)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Error: expected zero or one positional argument (results directory)")
		fmt.Print(usage)
		os.Exit(1)
	}

	dir := config.Dir
	if dir == "" {
		if len(args) == 1 {
			dir = args[0]
		} else {
			dir = "."
		}
	}
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.ScanInterval <= 0 {
		config.ScanInterval = 2 * time.Second
	}

	runStore := store.NewRunStore(dir, config.ScanInterval)
	runStore.Start()
	server := backend.NewServer(config.Addr, runStore)
	fmt.Printf("Serving on http://localhost%s (watching %s)\n", normalizeLocalhostAddr(config.Addr), dir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs() *Config {
	config := &Config{}

	flag.BoolVar(&config.Help, "h", false, "Show help")
	flag.BoolVar(&config.Help, "help", false, "Show help")
	flag.StringVar(&config.Dir, "dir", "", "Directory to scan for Robot XML outputs")
	flag.StringVar(&config.Addr, "addr", ":8080", "HTTP listen address")
	flag.DurationVar(&config.ScanInterval, "scan-interval", 2*time.Second, "Directory scan interval")

	flag.Usage = func() {
		fmt.Print(usage)
	}

	flag.Parse()
	return config
}

func normalizeLocalhostAddr(addr string) string {
	// Print a friendly localhost URL for default cases.
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	return addr
}
