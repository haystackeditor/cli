package compact

import "encoding/json"

const (
	// maxCompactStringBytes caps individual tool output/input string values in
	// compacted output. Large tool I/O — a big command dump, a Write tool's
	// whole file body, a long plan — carries little parser-relevant signal but
	// bloats full.jsonl well past the ~30-200KB target for long sessions.
	maxCompactStringBytes = 8 * 1024
	compactTruncMarker    = "…[truncated]"
)

// truncateCompactString shortens s beyond maxCompactStringBytes, appending a
// marker. Small strings are returned unchanged.
func truncateCompactString(s string) string {
	if len(s) <= maxCompactStringBytes {
		return s
	}
	return s[:maxCompactStringBytes] + compactTruncMarker
}

// truncateInputStrings walks a tool_use "input" value and truncates any
// oversized string while preserving structure. Best-effort: it returns the
// input unchanged on any parse/marshal problem (and for small inputs), so it is
// byte-identical for the common case and can never fail compaction.
func truncateInputStrings(input json.RawMessage) json.RawMessage {
	if len(input) <= maxCompactStringBytes {
		return input
	}
	var v any
	if json.Unmarshal(input, &v) != nil {
		return input
	}
	out, err := json.Marshal(truncateJSONValue(v))
	if err != nil {
		return input
	}
	return out
}

// truncateJSONValue recursively truncates oversized strings within a decoded
// JSON value, leaving structure intact.
func truncateJSONValue(v any) any {
	switch t := v.(type) {
	case string:
		return truncateCompactString(t)
	case []any:
		for i := range t {
			t[i] = truncateJSONValue(t[i])
		}
		return t
	case map[string]any:
		for k, val := range t {
			t[k] = truncateJSONValue(val)
		}
		return t
	default:
		return v
	}
}
