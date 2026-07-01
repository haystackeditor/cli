package compact

import (
	"bufio"
	"bytes"
	"encoding/json"
)

const (
	// fallbackMaxLineBytes bounds a single JSONL line in the fallback output.
	fallbackMaxLineBytes = 16 * 1024
	// fallbackMaxTotalBytes bounds the whole fallback transcript (~1 MiB).
	fallbackMaxTotalBytes = 1 << 20
)

// fallbackHeavyFields are large Claude Code tool-result payloads that carry no
// parser-relevant signal but dominate raw transcript size — e.g. a single
// Edit's originalFile can be hundreds of KB.
var fallbackHeavyFields = []string{"originalFile", "oldString", "newString", "structuredPatch"}

// SafeFallback produces a bounded, best-effort transcript for use when Compact
// returns an error or an empty result. It never errors and never returns a
// large blob: it strips the known heavy tool-result fields line-by-line and
// caps both per-line and total size. This guarantees that a compaction failure
// can never cause a multi-MB raw transcript to be written to the
// entire/checkpoints/v1 branch (previously the fallback wrote the full raw
// transcript, chunked at 50 MiB, permanently bloating the branch).
func SafeFallback(content []byte) []byte {
	var out bytes.Buffer
	r := bufio.NewReader(bytes.NewReader(content))
	for out.Len() < fallbackMaxTotalBytes {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			out.Write(fallbackShrinkLine(line))
		}
		if err != nil { // io.EOF or any read error: stop cleanly
			break
		}
	}
	return out.Bytes()
}

// fallbackShrinkLine returns a size-bounded version of a single JSONL line.
// Small lines pass through unchanged. Oversized lines have heavy tool-result
// fields removed; if a line is still too large (or unparseable), it is replaced
// with a compact, valid marker object so downstream JSONL parsers stay happy.
func fallbackShrinkLine(line []byte) []byte {
	trimmed := bytes.TrimRight(line, "\r\n")
	if len(trimmed) <= fallbackMaxLineBytes {
		return fallbackEnsureNewline(line)
	}

	var raw map[string]json.RawMessage
	if json.Unmarshal(trimmed, &raw) != nil {
		return fallbackMarker("")
	}

	if turRaw, ok := raw["toolUseResult"]; ok {
		var tur map[string]json.RawMessage
		if json.Unmarshal(turRaw, &tur) == nil {
			for _, f := range fallbackHeavyFields {
				delete(tur, f)
			}
			if nb, err := json.Marshal(tur); err == nil {
				raw["toolUseResult"] = nb
			} else {
				delete(raw, "toolUseResult")
			}
		} else {
			delete(raw, "toolUseResult")
		}
	}

	nb, err := json.Marshal(raw)
	if err != nil || len(nb) > fallbackMaxLineBytes {
		var kind string
		if t, ok := raw["type"]; ok {
			_ = json.Unmarshal(t, &kind)
		}
		return fallbackMarker(kind)
	}
	return append(nb, '\n')
}

// fallbackMarker returns a small valid JSONL line standing in for a dropped
// oversized entry, preserving the entry kind when known.
func fallbackMarker(kind string) []byte {
	if kind == "" {
		kind = "unknown"
	}
	b, _ := json.Marshal(map[string]any{"type": kind, "_truncated": true})
	return append(b, '\n')
}

func fallbackEnsureNewline(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		return line
	}
	return append(line, '\n')
}
