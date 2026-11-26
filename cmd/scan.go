package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"acovia.net/mscan/scan"
	"acovia.net/record"
)

func scanCmd() error {
	const usage = "mscan scan <host>"
	if err := checkArgs(usage, 1); err != nil {
		return err
	}

	host := strings.Trim(subCmdArgs[0], "[]")
	addresses, err := net.LookupHost(host)
	if err != nil {
		return record.FmtError("resolve host:", err)
	}
	address := addresses[0]

	ports, err := parsePorts()
	if err != nil {
		return err
	}
	workers, err := parseWorkers()
	if err != nil {
		return err
	}
	timeout, err := parseTimeout(defaultScanTimeout)
	if err != nil {
		return err
	}
	rate, err := parseRate()
	if err != nil {
		return err
	}

	rateText := fmt.Sprintf("%d/s", rate)
	if rate == 0 {
		rateText = "unlimited"
	}
	record.Info("scanning", host, "("+address+")", "ports", portsFlag, "workers", workers, "rate", rateText, "timeout", timeout)

	start := time.Now()
	open, minecraft := 0, 0
	results := scan.Scan(scan.Options{
		Address: address,
		Host:    host,
		Ports:   ports,
		Workers: workers,
		Timeout: timeout,
		Rate:    rate,
	})
	for result := range results {
		open++
		if result.Status == nil {
			if !onlyMC {
				record.Printf("%v/tcp\topen\n", result.Port)
			}
			continue
		}
		minecraft++
		record.Printf("tcp/%v\tminecraft\t%v\t%v/%v\t%v\t%v\n",
			result.Port,
			result.Status.Online,
			result.Status.Max,
			result.Status.Version,
			oneLine(result.Status.MOTD),
			formatLatency(result.Status.Latency),
		)
	}

	record.Info(fmt.Sprintf("scan finished in %v: %d open, %d minecraft",
		time.Since(start).Round(time.Second), open, minecraft))
	return nil
}

// oneLine collapses whitespace so a MOTD fits in one output line.
func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
