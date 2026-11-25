package main

import "acovia.net/record"

func help() error {
	record.Print(helpInfo)
	return nil
}

var helpInfo = `mscan [option] <command> [args...]

commands:

	scan <host>
		scan the TCP ports of <host> and identify Minecraft servers.
		ports defaults to 1-65535, workers to 256 and the timeout to 1s.
		only Minecraft servers are printed unless -a is given.

	ping <host>[:port]
		query the status of one Minecraft server.
		without a colon, _minecraft._tcp.<host> is looked up first and
		port 25565 is used when there is no SRV record.
		with a colon the port is used as given and SRV is skipped.

	help
		print this help text.

	about
		print the name, repository and license of this tool.

options:

	-t --timeout <duration>:
		connect and probe timeout, in Go duration syntax such as 500ms,
		2s or 1m. defaults to 5s for ping and 1s for scan.

	-w --workers <number>:
		number of ports scanned concurrently. defaults to 256.
		a higher value is faster but can miss ports on hosts that
		rate-limit connection bursts.

	-p --ports <specification>:
		ports to scan: 25565, 1-1024, or a comma separated list such as
		1-100,25565. defaults to 1-65535.

	-o --only-mc:
		only print mc open port.

	--help:
		print this help text.

`
