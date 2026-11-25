package ping

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// maxStatusPacket bounds the status JSON so a broken server cannot make the
// client allocate an arbitrary amount of memory.
const maxStatusPacket = 2 << 20

// ProbeConn sends a status request over an established connection. The
// caller is responsible for setting a deadline on conn.
func ProbeConn(conn net.Conn, host string, port int) (*Status, error) {
	start := time.Now()
	if _, err := conn.Write(buildRequest(host, port)); err != nil {
		return nil, fmt.Errorf("write handshake: %w", err)
	}

	length, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("read status length: %w", err)
	}
	if length <= 0 || length > maxStatusPacket {
		return nil, fmt.Errorf("invalid status packet length: %d", length)
	}

	packet := make([]byte, length)
	if _, err := io.ReadFull(conn, packet); err != nil {
		return nil, fmt.Errorf("read status packet: %w", err)
	}

	reader := bytes.NewReader(packet)
	packetID, err := readVarInt(reader)
	if err != nil {
		return nil, fmt.Errorf("read packet id: %w", err)
	}
	if packetID != 0 {
		return nil, fmt.Errorf("unexpected packet id: %d", packetID)
	}
	jsonLength, err := readVarInt(reader)
	if err != nil || jsonLength < 0 || jsonLength > reader.Len() {
		return nil, errors.New("invalid status JSON length")
	}
	document := make([]byte, jsonLength)
	if _, err := io.ReadFull(reader, document); err != nil {
		return nil, fmt.Errorf("read status JSON: %w", err)
	}

	var response struct {
		Version struct {
			Name string `json:"name"`
		} `json:"version"`
		Players struct {
			Online int `json:"online"`
			Max    int `json:"max"`
			Sample []struct {
				Name string `json:"name"`
			} `json:"sample"`
		} `json:"players"`
		Description json.RawMessage `json:"description"`
	}
	if err := json.Unmarshal(document, &response); err != nil {
		return nil, fmt.Errorf("parse status JSON: %w", err)
	}

	status := &Status{
		Version: strings.TrimSpace(response.Version.Name),
		Online:  max(0, response.Players.Online),
		Max:     max(0, response.Players.Max),
		MOTD:    flattenDescription(response.Description),
		Latency: time.Since(start),
	}
	if status.Version == "" {
		status.Version = "unknown"
	}
	for _, player := range response.Players.Sample {
		if name := strings.TrimSpace(player.Name); name != "" {
			status.Sample = append(status.Sample, name)
		}
	}
	return status, nil
}

// buildRequest returns the handshake and status request packets.
func buildRequest(host string, port int) []byte {
	payload := make([]byte, 0, 32+len(host))
	payload = appendVarInt(payload, 0)  // packet id: handshake
	payload = appendVarInt(payload, -1) // protocol version: -1 for status only
	payload = appendVarInt(payload, len(host))
	payload = append(payload, host...)
	payload = append(payload, byte(port>>8), byte(port))
	payload = appendVarInt(payload, 1) // next state: status

	packet := appendVarInt(nil, len(payload))
	packet = append(packet, payload...)
	return append(packet, 0x01, 0x00) // status request
}

// flattenDescription turns a chat component into plain text.
func flattenDescription(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	var builder strings.Builder
	writeDescription(&builder, value)
	description := strings.TrimSpace(stripFormatting(builder.String()))
	if description == "" {
		description = "null"
	}

	return description
}

func writeDescription(builder *strings.Builder, value any) {
	switch v := value.(type) {
	case string:
		builder.WriteString(v)
	case []any:
		for _, item := range v {
			writeDescription(builder, item)
		}
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			builder.WriteString(text)
		}
		if extra, ok := v["extra"].([]any); ok {
			for _, item := range extra {
				writeDescription(builder, item)
			}
		}
	}
}

// stripFormatting removes legacy § formatting codes and normalizes whitespace.
func stripFormatting(value string) string {
	var builder strings.Builder
	skip := false
	for _, char := range value {
		switch {
		case char == '§':
			skip = true
		case skip:
			skip = false
		case char == '\r':
		case char == '\t':
			builder.WriteByte(' ')
		default:
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

// appendVarInt appends a Minecraft protocol VarInt.
func appendVarInt(dst []byte, value int) []byte {
	number := uint32(value)
	for {
		current := byte(number & 0x7f)
		number >>= 7
		if number != 0 {
			current |= 0x80
		}
		dst = append(dst, current)
		if number == 0 {
			return dst
		}
	}
}

// readVarInt reads a Minecraft protocol VarInt (at most 5 bytes).
func readVarInt(reader io.Reader) (int, error) {
	var value uint32
	for index := 0; index < 5; index++ {
		var buffer [1]byte
		if _, err := io.ReadFull(reader, buffer[:]); err != nil {
			return 0, err
		}
		value |= uint32(buffer[0]&0x7f) << (7 * index)
		if buffer[0]&0x80 == 0 {
			return int(value), nil
		}
	}
	return 0, errors.New("VarInt too long")
}
