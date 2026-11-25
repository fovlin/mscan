# Develop docs

## Layout

```text
mscan
├── go.mod     module acovia.net/mscan
├── go.sum
├── cmd/       CLI
│   ├── main.go      command table, option registration, dispatch
│   ├── init.go      shared state, -t/-w/-r/-p parsing, checkArgs, latency formatting
│   ├── help.go      usage text
│   ├── about.go     about text
│   ├── ping.go      ping command
│   ├── scan.go      scan command
│   └── init_test.go
├── ping/      status ping, package acovia.net/mscan/ping
│   ├── ping.go      Resolve, Ping, ProbeConn
│   ├── proto.go     VarInts, request building, status parsing, MOTD flattening
│   └── ping_test.go
├── scan/      TCP port scanner, package acovia.net/mscan/scan
│   ├── scan.go      fixed worker pool and rate limiter
│   └── scan_test.go
├── build.sh
├── README.md
└── DEVELOP.md
```

`cmd` imports `github.com/fovlin/opt` for option parsing and
`acovia.net/record` for logging. `acovia.net/record` is not published yet, so
the root `go.mod` replaces it with `../mc-saver/record`; everything else is a
package of this module.

Build and run from the repository root:

```bash
go build -o mscan ./cmd
./mscan help
```

## Command layer

Commands are registered once:

```go
var commands = map[string]func() error{
	"scan":  scanCmd,
	"ping":  pingCmd,
	"help":  help,
	"about": about,
}
```

`initProgram` registers the options, calls `opt.Parse`, then stores the command
name in `cmd` and the remaining words in `subCmdArgs`. With no arguments the
command defaults to `help`. Handlers validate their own argument count with
`checkArgs`. `main` prints errors with `record.Error`, which exits with status
1, so handlers only return errors.

## ping

`Resolve` implements the SRV rule: a target with a colon is used as host:port;
an IP literal gets port 25565; a hostname is looked up as
`_minecraft._tcp.<host>` with port 25565 as the fallback. The handshake host is
the SRV target when SRV was used.

`Ping` dials the resolved address, sets a deadline, and calls `ProbeConn`,
which sends a handshake with protocol version -1 plus a status request, then
parses the status packet (capped at 2 MiB). The MOTD is flattened from the
chat component and legacy § formatting codes are stripped.

## scan

`Scan` streams open ports through a fixed worker pool: ports arrive on an
unbuffered channel, workers dial and probe, results are sent as they complete,
and a `WaitGroup` closes the result channel. A successful dial is always
reported; `Status` is nil when the probe failed, meaning the port is open but
not a Minecraft server.

`scanCmd` resolves the host once with `net.LookupHost`, dials the resolved IP
with `net.JoinHostPort`, and sends the original hostname in the handshake so
servers behind Velocity or BungeeCord route the status request by hostname.
The command prints every open port; `-o/--only-mc` filters the output to
Minecraft servers. The summary always counts all open ports.

The default of 256 workers and 128 connections per second is deliberate.
Targets that rate-limit connection bursts (game panels, home routers, DDoS
filters) drop concurrent dials, so a real open port can fail to connect and be
reported as closed, and a sustained scan can get the source IP banned. Tests on
a Pterodactyl-style host showed that 1024 workers found about half of its open
ports and triggered an IP ban, while 256 workers found all of them. The rate
limiter in `Scan` paces new connections with a ticker in the port producer;
raise or lower `-w` and `-r` with that trade-off in mind.

## Release

1. Run `gofmt -l .`, `go vet ./...` and `go test ./...` at the repository root.
2. `go build -o mscan ./cmd` and smoke-test `ping`, `scan`, `help` and `about`.
3. Tag `vX.Y.Z` and push, then run `./build.sh` and attach the archives under
   `build/` to the release.
