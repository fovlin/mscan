# mscan

A small Minecraft server ping and TCP port scan tool.

## Build

Go 1.27 or newer. Build and run from the repository root:

```bash
go build -o mscan ./cmd
./mscan help
```

`github.com/fovlin/opt` is fetched from the module proxy and pinned in
`cmd/go.sum`. `acovia.net/record` is not published yet, so `cmd/go.mod`
resolves it from a sibling mc-saver checkout:

```go
replace acovia.net/record => ../../mc-saver/record
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
| `-p`, `--ports <spec>` | Ports to scan: `25565`, `1-1024`, or `1-100,25565`. Defaults to `1-65535`. |
| `-a`, `--all-ports` | Print every open port, not only Minecraft servers. |
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
[INFO]: no SRV record, using port 25565
[INFO]: ping mc.acovia.net (mc.acovia.net:25565)
[INFO]: version: Velocity 1.7.2-26.3
[INFO]: players: 1/500
[INFO]: sample: JesusOnlyFans
[INFO]: motd: Baka Cloud
[INFO]: latency: 61ms
```

```bash
$ mscan ping mc.example.com:25565
[INFO]: explicit port given, SRV lookup skipped
[INFO]: ping mc.example.com:25565 (mc.example.com:25565)
[INFO]: version: 1.21.4
[INFO]: players: 3/20
[INFO]: motd: Example server
[INFO]: latency: 18ms
```

```bash
$ mscan scan mc.example.com -p 25565-25567 -t 2s
[INFO]: scanning mc.example.com (1.2.3.4) ports 25565-25567 workers 256 timeout 2s
25565/tcp	minecraft	1.21.4	3/20	Example server	18ms
[INFO]: scan finished in 2s: 1 open, 1 minecraft
```

Only Minecraft servers are printed by default. With `-a/--all-ports`, every
open port is printed as `<port>/tcp	open` as well. With the defaults, a full
scan of a filtered host takes about 256 seconds (65535 ports / 256 workers ×
1s timeout).

The default worker count is deliberately conservative. Hosts that rate-limit
connection bursts (game panels, home routers, DDoS filters) can drop concurrent
dials, which makes real open ports look closed. A higher `-w` only speeds up
well-behaved hosts; lower it further, for example `-w 128`, when a full scan of
a protected host still misses ports.

## License

GNU General Public License v3.
