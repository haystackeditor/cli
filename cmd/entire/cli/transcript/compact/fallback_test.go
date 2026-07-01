package compact

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// A raw Claude Code Edit tool-result line embeds the entire pre-edit file in
// toolUseResult.originalFile; SafeFallback must strip it and stay small.
func TestSafeFallback_StripsHeavyToolResult(t *testing.T) {
	big := strings.Repeat("x", 600*1024)
	line, err := json.Marshal(map[string]any{
		"type": "user",
		"toolUseResult": map[string]any{
			"type":         "text",
			"originalFile": big,
			"file":         map[string]any{"filePath": "/repo/a.go", "numLines": 3},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := SafeFallback(append(line, '\n'))

	if len(out) > 32*1024 {
		t.Fatalf("expected bounded output, got %d bytes", len(out))
	}
	if bytes.Contains(out, []byte("originalFile")) {
		t.Fatalf("originalFile should have been stripped")
	}
	for _, l := range bytes.Split(bytes.TrimSpace(out), []byte("\n")) {
		var m map[string]json.RawMessage
		if json.Unmarshal(l, &m) != nil {
			t.Fatalf("fallback produced invalid JSON line: %s", l)
		}
	}
}

// A pathologically large transcript must yield a bounded fallback, never the
// full raw content.
func TestSafeFallback_TotalCap(t *testing.T) {
	var buf bytes.Buffer
	line, _ := json.Marshal(map[string]any{"type": "assistant", "message": strings.Repeat("y", 20*1024)})
	for i := 0; i < 1000; i++ { // ~20 MB of oversized lines
		buf.Write(line)
		buf.WriteByte('\n')
	}
	out := SafeFallback(buf.Bytes())
	if len(out) > fallbackMaxTotalBytes+fallbackMaxLineBytes {
		t.Fatalf("expected bounded total (~%d), got %d bytes", fallbackMaxTotalBytes, len(out))
	}
}

// Small lines must pass through unchanged so a fallback stays useful to
// downstream consumers.
func TestSafeFallback_SmallLinesUnchanged(t *testing.T) {
	in := []byte(`{"type":"user","message":"hi"}` + "\n" + `{"type":"assistant","message":"ok"}` + "\n")
	out := SafeFallback(in)
	if !bytes.Equal(bytes.TrimSpace(out), bytes.TrimSpace(in)) {
		t.Fatalf("small lines should pass through unchanged:\n got: %s\nwant: %s", out, in)
	}
}

// An unparseable oversized line becomes a valid marker, never raw bloat.
func TestSafeFallback_UnparseableOversizedLine(t *testing.T) {
	out := SafeFallback([]byte(strings.Repeat("z", 100*1024) + "\n"))
	if len(out) > 1024 {
		t.Fatalf("expected tiny marker, got %d bytes", len(out))
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(out), &m) != nil {
		t.Fatalf("marker must be valid JSON: %s", out)
	}
}
