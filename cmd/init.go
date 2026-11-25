package main

import (
	"strconv"
	"strings"
	"time"

	"acovia.net/record"
)

var (
	timeoutFlag string
	workersFlag string
	portsFlag   string
	onlyMC      bool

	defaultPingTimeout = 5 * time.Second
	defaultScanTimeout = time.Second
)

// parseTimeout returns the -t value, or fallback when the flag is absent.
func parseTimeout(fallback time.Duration) (time.Duration, error) {
	if timeoutFlag == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(timeoutFlag)
	if err != nil {
		return 0, record.FmtError("parse timeout:", err)
	}
	return value, nil
}

// parseWorkers returns the -w value.
func parseWorkers() (int, error) {
	value, err := strconv.Atoi(workersFlag)
	if err != nil || value < 1 {
		return 0, record.FmtError("invalid workers:", workersFlag)
	}
	return value, nil
}

// parsePorts parses the -p value, for example "25565" or "1-1024,25565".
func parsePorts() ([]int, error) {
	var result []int
	for _, part := range strings.Split(portsFlag, ",") {
		firstPart, lastPart, hasRange := strings.Cut(part, "-")
		first, err := parsePort(firstPart)
		if err != nil {
			return nil, err
		}
		last := first
		if hasRange {
			if last, err = parsePort(lastPart); err != nil {
				return nil, err
			}
		}
		if last < first {
			return nil, record.FmtError("invalid port range:", part)
		}
		for port := first; port <= last; port++ {
			result = append(result, port)
		}
	}
	return result, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, record.FmtError("invalid port:", value)
	}
	return port, nil
}

// formatLatency rounds a latency for display, keeping sub-millisecond values
// readable for loopback targets.
func formatLatency(value time.Duration) time.Duration {
	if value < time.Millisecond {
		return value.Round(time.Microsecond)
	}
	return value.Round(time.Millisecond)
}

// checkArgs reports a usage error unless the command got want arguments.
func checkArgs(usage string, want int) error {
	if len(subCmdArgs) != want {
		return record.FmtError("usage:", usage)
	}
	return nil
}
