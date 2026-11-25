# mscan

A small Minecraft server ping and TCP port scan tool.

## Build

Go 1.27 or newer. Build and run from the repository root:

```bash
go build -o mscan ./cmd
./mscan help
```

`github.com/fovlin/opt` is fetched from the module proxy and pinned in
`go.sum`. `acovia.net/record` is not published yet, so `go.mod` resolves it
from a sibling mc-saver checkout:

```go
replace acovia.net/record => ../mc-saver/record
```

Adjust that path, or drop the replace once the module is available, before
building elsewhere.

## Usage

```text
mscan [option] <command> [args...]
```

| Command | Description |
| --- | --- |
| `scan <host>` | Scan ports and identify Minecraft servers. |
| `ping <host>[:port]` | Print the status of one Minecraft server. |
| `help`, `about` | Print the usage text or information about this tool. |

| Flag | Description |
| --- | --- |
| `-t`, `--timeout <duration>` | Connect and probe timeout. Defaults to 5s for ping and 1s for scan. |
| `-w`, `--workers <number>` | Number of ports scanned concurrently. Defaults to 256. |
| `-r`, `--rate <number>` | Maximum new connections per second. Defaults to 128; `0` disables pacing. |
| `-p`, `--ports <spec>` | Ports to scan: `25565`, `1-1024`, or `1-100,25565`. Defaults to `1-65535`. |
| `-o`, `--only-mc` | Print only Minecraft servers, not every open port. |
| `--help` | Print the usage text. |

### SRV rule

A target without a colon is a hostname: mscan looks up
`_minecraft._tcp.<host>` and uses the first SRV record, falling back to port
25565 when there is none. A target with a colon already carries its port, so
the SRV lookup is skipped. IPv6 literals are accepted as `2001:db8::1` or
`[2001:db8::1]:25565`.

### Examples

```bash
$ mscan ping mc.acovia.net
[INFO]: ping mc.acovia.net (mc.acovia.net:25565)
[INFO]: version: Velocity 1.7.2-26.3
[INFO]: players: 1/500
[INFO]: sample: JesusOnlyFans
[INFO]: motd: Baka Cloud
[INFO]: latency: 61ms
```

```bash
$ mscan ping mc.example.com:25565
[INFO]: ping mc.example.com:25565 (mc.example.com:25565)
[INFO]: version: 1.21.4
[INFO]: players: 3/20
[INFO]: motd: Example server
[INFO]: latency: 18ms
```

When an SRV record is used, an `SRV -> <target>:<port>` line is printed before
the status.

```bash
$ mscan scan mc.example.com -p 25565-25567 -t 2s
[INFO]: scanning mc.example.com (1.2.3.4) ports 25565-25567 workers 256 rate 128/s timeout 2s
25565/tcp	minecraft	1.21.4	3/20	Example server	18ms
[INFO]: scan finished in 2s: 1 open, 1 minecraft
```

Every open port is printed by default; `-o/--only-mc` limits the output to
Minecraft servers. With the defaults (128 connections per second), a full scan
takes about 510 seconds.

The default worker count and rate are deliberately conservative. Hosts that
rate-limit connection bursts (game panels, home routers, DDoS filters) can drop
concurrent dials, which makes real open ports look closed, and sustained scans
can get the source IP banned. Lower `-w` or `-r` when a full scan of a protected
host still misses ports.

### IP bans

A full-range scan opens up to 65535 connections. Hosts with anti-scan
protection may drop them or ban the source IP for a while, and repeated scans
make it worse. To avoid it:

- scan a narrow range with `-p`, for example the port the server actually uses;
- keep the rate low, for example `-r 50`, and avoid back-to-back full scans;
- if you own the host, whitelist the scanning IP in its firewall or panel:
  `fail2ban` `ignoreip`, CSF `csf.allow`, UFW `allow from <ip>`, or the
  provider's DDoS panel allowlist;
- if you own the host, scanning it locally (ssh in, then `mscan scan
  127.0.0.1`) never touches the firewall and cannot trigger a ban.

A ban is temporary in most panels; wait it out instead of scanning again.

## License

GNU General Public License v3.
