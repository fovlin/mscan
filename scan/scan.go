// Package scan probes TCP ports for Minecraft servers.
package scan

import (
	"net"
	"strconv"
	"sync"
	"time"

	"acovia.net/mscan/ping"
)

// Options configures a scan.
type Options struct {
	Address string // IP or host to dial
	Host    string // hostname sent in the Minecraft handshake
	Ports   []int
	Workers int
	Timeout time.Duration
	Rate    int // maximum new connections per second, 0 means no limit
}

// Result is an open port. Status is nil when the port is not a Minecraft
// server.
type Result struct {
	Port   int
	Status *ping.Status
}

// Scan dials every port with a fixed worker pool and streams the open ones.
func Scan(opts Options) <-chan Result {
	ports := make(chan int)
	results := make(chan Result)

	var waitGroup sync.WaitGroup
	for range opts.Workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for port := range ports {
				if result, ok := probe(opts, port); ok {
					results <- result
				}
			}
		}()
	}

	// Pace new connections so a scan is not mistaken for a SYN flood and does
	// not get the source IP banned by the target's anti-scan protection.
	go func() {
		defer close(ports)
		var ticker *time.Ticker
		if opts.Rate > 0 {
			interval := time.Second / time.Duration(opts.Rate)
			if interval <= 0 {
				interval = time.Nanosecond
			}
			ticker = time.NewTicker(interval)
			defer ticker.Stop()
		}
		for _, port := range opts.Ports {
			if ticker != nil {
				<-ticker.C
			}
			ports <- port
		}
	}()

	go func() {
		waitGroup.Wait()
		close(results)
	}()

	return results
}

// probe reports a result when the port accepted a connection.
func probe(opts Options, port int) (Result, bool) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(opts.Address, strconv.Itoa(port)), opts.Timeout)
	if err != nil {
		return Result{}, false
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(opts.Timeout)); err != nil {
		return Result{}, false
	}
	status, err := ping.ProbeConn(conn, opts.Host, port)
	if err != nil {
		return Result{Port: port}, true
	}
	return Result{Port: port, Status: status}, true
}
