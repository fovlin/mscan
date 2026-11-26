// Package ping performs Minecraft server list pings.
package ping

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// DefaultPort is the Minecraft port used when a target has no port.
const DefaultPort = 25565

// Status is a parsed server list response.
type Status struct {
	Version string
	Online  int
	Max     int
	Sample  []string
	MOTD    string
	Latency time.Duration
}

// Target is a resolved ping target.
type Target struct {
	Host    string // hostname sent in the handshake
	Port    int
	Address string // dialable host:port
	SRV     bool   // the port came from an SRV record
}

// lookupSRV is a variable so tests can avoid real DNS lookups.
var lookupSRV = net.LookupSRV

// Resolve turns target into a dialable address. A target with a colon is
// treated as host:port and skips SRV; a target without a colon is looked up
// as _minecraft._tcp.<host> and falls back to port 25565.
func Resolve(target string) (Target, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return Target{}, errors.New("empty target")
	}

	if host, portString, err := net.SplitHostPort(target); err == nil {
		port, err := strconv.Atoi(portString)
		if err != nil || port < 1 || port > 65535 {
			return Target{}, fmt.Errorf("invalid port %q", portString)
		}
		return newTarget(host, port, false), nil
	}

	host := strings.Trim(strings.TrimSuffix(target, "."), "[]")
	if ip := net.ParseIP(host); ip != nil {
		return newTarget(host, DefaultPort, false), nil
	}
	if strings.Contains(host, ":") {
		return Target{}, fmt.Errorf("invalid address %q", target)
	}

	_, records, err := lookupSRV("minecraft", "tcp", host)
	if err == nil && len(records) > 0 {
		srvHost := strings.TrimSuffix(records[0].Target, ".")
		return newTarget(srvHost, int(records[0].Port), true), nil
	}
	return newTarget(host, DefaultPort, false), nil
}

func newTarget(host string, port int, srv bool) Target {
	return Target{
		Host:    host,
		Port:    port,
		Address: net.JoinHostPort(host, strconv.Itoa(port)),
		SRV:     srv,
	}
}

// Ping dials a resolved target and returns its status.
func Ping(target Target, timeout time.Duration) (*Status, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", target.Address, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target.Address, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	status, err := ProbeConn(conn, target.Host, target.Port)
	if err != nil {
		return nil, err
	}

	status.Latency = time.Since(start)
	return status, nil
}
