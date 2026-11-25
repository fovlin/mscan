package scan

import (
	"net"
	"testing"
	"time"

	"acovia.net/mscan/ping"
)

// startServer starts a local TCP server. A Minecraft server answers the
// status request; any other server reads one chunk and closes.
func startServer(t *testing.T, minecraft bool) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				buffer := make([]byte, 512)
				_ = conn.SetReadDeadline(time.Now().Add(time.Second))
				if _, err := conn.Read(buffer); err != nil {
					return
				}
				if !minecraft {
					return
				}
				const document = `{"version":{"name":"Test"},"players":{"online":1,"max":2},"description":"hi"}`
				packet := append([]byte{0, byte(len(document))}, document...)
				_, _ = conn.Write(append([]byte{byte(len(packet))}, packet...))
			}(conn)
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port
}

// closedPort returns a port that nothing listens on any more.
func closedPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func TestScanRate(t *testing.T) {
	closed := closedPort(t)
	start := time.Now()
	for range Scan(Options{
		Address: "127.0.0.1",
		Host:    "127.0.0.1",
		Ports:   []int{closed, closed + 1, closed + 2},
		Workers: 3,
		Timeout: time.Second,
		Rate:    10,
	}) {
	}
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Fatalf("scan was not paced: %v", elapsed)
	}
}

func TestScan(t *testing.T) {
	minecraftPort := startServer(t, true)
	openPort := startServer(t, false)
	closed := closedPort(t)

	results := Scan(Options{
		Address: "127.0.0.1",
		Host:    "127.0.0.1",
		Ports:   []int{minecraftPort, openPort, closed},
		Workers: 3,
		Timeout: time.Second,
	})

	found := make(map[int]*ping.Status)
	for result := range results {
		found[result.Port] = result.Status
	}

	if len(found) != 2 {
		t.Fatalf("got %d open ports, want 2: %v", len(found), found)
	}
	status, ok := found[minecraftPort]
	if !ok || status == nil {
		t.Fatalf("minecraft port not detected: %v", found)
	}
	if status.Version != "Test" || status.Online != 1 || status.Max != 2 || status.MOTD != "hi" {
		t.Fatalf("minecraft status: %+v", status)
	}
	if status, ok := found[openPort]; !ok || status != nil {
		t.Fatalf("plain port: %v %v", ok, status)
	}
}
