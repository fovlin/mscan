package main

import (
	"fmt"
	"strings"

	"acovia.net/mscan/ping"
	"acovia.net/record"
)

func pingCmd() error {
	const usage = "mscan ping <host>[:port]"
	if err := checkArgs(usage, 1); err != nil {
		return err
	}

	target, err := ping.Resolve(subCmdArgs[0])
	if err != nil {
		return record.FmtError("resolve target:", err)
	}

	if target.SRV {
		record.Info("SRV ->", target.Address)
	}
	record.Info("ping", subCmdArgs[0], "("+target.Address+")")

	timeout, err := parseTimeout(defaultPingTimeout)
	if err != nil {
		return err
	}
	status, err := ping.Ping(target, timeout)
	if err != nil {
		return record.FmtError("ping", target.Address+":", err)
	}

	record.Info("version:", status.Version)
	record.Info("players:", fmt.Sprintf("%d/%d", status.Online, status.Max))
	if len(status.Sample) > 0 {
		record.Info("sample:", strings.Join(status.Sample, ", "))
	}
	record.Info("motd:", status.MOTD)
	record.Info("latency:", formatLatency(status.Latency))
	return nil
}
