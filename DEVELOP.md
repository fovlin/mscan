# Develop docs

## Layout

```text
mscan
├── cmd/     CLI, module acovia.net/mscan
│   ├── main.go      command table, option registration, dispatch
│   ├── init.go      shared state, -t/-w/-p parsing, checkArgs, latency formatting
│   ├── help.go      usage text
│   ├── about.go     about text
│   ├── ping.go      ping command
│   ├── scan.go      scan command
│   └── init_test.go
├── ping/    status ping, module acovia.net/mscan/ping
│   ├── ping.go      Resolve, Ping, ProbeConn
│   ├── proto.go     VarInts, request building, status parsing, MOTD flattening
│   └── ping_test.go
├── scan/    TCP port scanner, module acovia.net/mscan/scan
│   ├── scan.go      fixed worker pool
│   └── scan_test.go
├── go.work
├── build.sh
├── README.md
└── DEVELOP.md
```

`cmd` imports `github.com/fovlin/opt` for option parsing and
`acovia.net/record` for logging. `acovia.net/record` is not published yet, so
`cmd/go.mod` replaces it with `../../mc-saver/record`; the other modules are
tied together with local replaces as well, and `go.work` lists the three
in-repo modules.

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
The command prints only Minecraft servers; `-a/--all-ports` also prints the
plain open ports, which are otherwise only counted in the summary.

The default of 256 workers is deliberate. Targets that rate-limit connection
bursts (game panels, home routers, DDoS filters) drop concurrent dials, so a
real open port can fail to connect and be reported as closed. Tests on a
Pterodactyl-style host showed that 1024 workers found about half of its open
ports, while 256 workers found all of them; raise or lower `-w` with that
trade-off in mind.

## Release

1. Run `gofmt -l`, `go vet ./...` and `go test ./...` in each module.
2. `go build -o mscan ./cmd` and smoke-test `ping`, `scan`, `help` and `about`.
3. Tag `vX.Y.Z` and push, then run `./build.sh` and attach the archives under
   `build/` to the release.
