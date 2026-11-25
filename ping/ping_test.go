package ping

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func readPacket(reader io.Reader) ([]byte, error) {
	length, err := readVarInt(reader)
	if err != nil {
		return nil, err
	}
	if length <= 0 || length > maxStatusPacket {
		return nil, errors.New("invalid packet length")
	}
	packet := make([]byte, length)
	if _, err := io.ReadFull(reader, packet); err != nil {
		return nil, err
	}
	return packet, nil
}

func statusResponse(document string) []byte {
	body := appendVarInt(nil, 0)
	body = appendVarInt(body, len(document))
	body = append(body, document...)
	return append(appendVarInt(nil, len(body)), body...)
}

// startStatusServer serves a status response to every connection.
func startStatusServer(t *testing.T, document string) string {
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
				if _, err := readPacket(conn); err != nil {
					return
				}
				if _, err := readPacket(conn); err != nil {
					return
				}
				_, _ = conn.Write(statusResponse(document))
			}(conn)
		}
	}()
	return listener.Addr().String()
}

func TestPing(t *testing.T) {
	address := startStatusServer(t, `{
		"version": {"name": "Test 1.20.4", "protocol": 765},
		"players": {"online": 2, "max": 10, "sample": [{"name": "alice"}, {"name": " bob "}, {"name": ""}]},
		"description": {"text": "Hello ", "extra": [{"text": "World"}]}
	}`)

	target := Target{Host: "mc.example.com", Port: 25565, Address: address}
	status, err := Ping(target, 2*time.Second)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if status.Version != "Test 1.20.4" {
		t.Fatalf("version = %q", status.Version)
	}
	if status.Online != 2 || status.Max != 10 {
		t.Fatalf("players = %d/%d", status.Online, status.Max)
	}
	if len(status.Sample) != 2 || status.Sample[0] != "alice" || status.Sample[1] != "bob" {
		t.Fatalf("sample = %v", status.Sample)
	}
	if status.MOTD != "Hello World" {
		t.Fatalf("motd = %q", status.MOTD)
	}
	if status.Latency <= 0 {
		t.Fatalf("latency = %v", status.Latency)
	}
}

func TestPingTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		<-release
	}()

	target := Target{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, Address: listener.Addr().String()}
	start := time.Now()
	if _, err := Ping(target, 100*time.Millisecond); err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}
}

func TestPingOversize(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = readPacket(conn)
		_, _ = readPacket(conn)
		_, _ = conn.Write(appendVarInt(nil, maxStatusPacket+1))
	}()

	target := Target{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, Address: listener.Addr().String()}
	_, err = Ping(target, 2*time.Second)
	if err == nil || !strings.Contains(err.Error(), "invalid status packet length") {
		t.Fatalf("expected a length error, got %v", err)
	}
}

func TestResolve(t *testing.T) {
	original := lookupSRV
	lookupSRV = func(service, proto, name string) (string, []*net.SRV, error) {
		switch name {
		case "mc.example.com":
			return "", []*net.SRV{{Target: "srv.example.com.", Port: 25566}}, nil
		case "plain.example.com":
			return "", nil, &net.DNSError{Err: "not found", Name: name, IsNotFound: true}
		}
		return "", nil, errors.New("unexpected lookup")
	}
	t.Cleanup(func() { lookupSRV = original })

	tests := []struct {
		name    string
		target  string
		host    string
		port    int
		address string
		srv     bool
		wantErr bool
	}{
		{"srv hit", "mc.example.com", "srv.example.com", 25566, "srv.example.com:25566", true, false},
		{"srv hit trailing dot", "mc.example.com.", "srv.example.com", 25566, "srv.example.com:25566", true, false},
		{"srv miss", "plain.example.com", "plain.example.com", 25565, "plain.example.com:25565", false, false},
		{"explicit port", "mc.example.com:1234", "mc.example.com", 1234, "mc.example.com:1234", false, false},
		{"ipv4", "1.2.3.4", "1.2.3.4", 25565, "1.2.3.4:25565", false, false},
		{"ipv4 port", "1.2.3.4:1234", "1.2.3.4", 1234, "1.2.3.4:1234", false, false},
		{"ipv6", "2001:db8::1", "2001:db8::1", 25565, "[2001:db8::1]:25565", false, false},
		{"ipv6 port", "[2001:db8::1]:25565", "2001:db8::1", 25565, "[2001:db8::1]:25565", false, false},
		{"bad port", "mc.example.com:abc", "", 0, "", false, true},
		{"zero port", "mc.example.com:0", "", 0, "", false, true},
		{"empty", "   ", "", 0, "", false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, err := Resolve(test.target)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", target)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if target.Host != test.host || target.Port != test.port || target.Address != test.address || target.SRV != test.srv {
				t.Fatalf("got %+v", target)
			}
		})
	}
}

func TestFlattenDescription(t *testing.T) {
	tests := []struct {
		document string
		want     string
	}{
		{`"hello"`, "hello"},
		{`{"text":"Hello ","extra":[{"text":"World"}]}`, "Hello World"},
		{`{"color":"#09ADD3","text":"Baka Cloud"}`, "Baka Cloud"},
		{`{"text":"§aGreen §rText"}`, "Green Text"},
		{`[{"text":"a"},{"text":"b"}]`, "ab"},
		{`null`, ""},
		{`{`, ""},
	}
	for _, test := range tests {
		if got := flattenDescription(jsonRaw(test.document)); got != test.want {
			t.Fatalf("%s: got %q, want %q", test.document, got, test.want)
		}
	}
}

func jsonRaw(document string) []byte {
	return []byte(document)
}

func TestVarInt(t *testing.T) {
	for _, value := range []int{0, 1, 127, 128, 255, 300, 16383, 1 << 20} {
		encoded := appendVarInt(nil, value)
		decoded, err := readVarInt(strings.NewReader(string(encoded)))
		if err != nil {
			t.Fatalf("%d: %v", value, err)
		}
		if decoded != value {
			t.Fatalf("%d: got %d", value, decoded)
		}
	}
	if length := len(appendVarInt(nil, -1)); length != 5 {
		t.Fatalf("-1 encoded to %d bytes, want 5", length)
	}
	if _, err := readVarInt(strings.NewReader("\xff\xff\xff\xff\xff\x01")); err == nil {
		t.Fatal("expected an error for an overlong VarInt")
	}
}
