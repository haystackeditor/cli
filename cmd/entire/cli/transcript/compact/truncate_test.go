package compact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTruncateCompactString(t *testing.T) {
	if got := truncateCompactString("short"); got != "short" {
		t.Fatalf("small string changed: %q", got)
	}
	big := strings.Repeat("a", maxCompactStringBytes+100)
	got := truncateCompactString(big)
	if len(got) >= len(big) || !strings.HasSuffix(got, compactTruncMarker) {
		t.Fatalf("large string not truncated (len=%d)", len(got))
	}
}

func TestTruncateInputStrings_SmallUnchanged(t *testing.T) {
	small := json.RawMessage(`{"file_path":"/a.go","content":"x"}`)
	if got := truncateInputStrings(small); string(got) != string(small) {
		t.Fatalf("small input should be byte-identical, got: %s", got)
	}
}

func TestTruncateInputStrings_LargeTruncatedStructureKept(t *testing.T) {
	big := strings.Repeat("z", maxCompactStringBytes+50)
	in, _ := json.Marshal(map[string]any{"file_path": "/a.go", "content": big})
	out := truncateInputStrings(in)

	var m map[string]string
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("truncated input is invalid JSON: %v", err)
	}
	if m["file_path"] != "/a.go" {
		t.Fatalf("structure not preserved: %v", m)
	}
	if len(m["content"]) >= len(big) || !strings.HasSuffix(m["content"], compactTruncMarker) {
		t.Fatalf("content not truncated (len=%d)", len(m["content"]))
	}
}
