package parser

import "testing"

func FuzzCleanURLTrailing(f *testing.F) {
	f.Add("https://example.com/path).")
	f.Add("http://example.com/test],")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		cleaned := CleanURLTrailing(input)
		if len(cleaned) > len(input) {
			t.Fatalf("cleaned URL grew: input=%q cleaned=%q", input, cleaned)
		}

		if CleanURLTrailing(cleaned) != cleaned {
			t.Fatalf("cleaned URL is not idempotent: %q", cleaned)
		}
	})
}

func FuzzBuildLineIndexAndOffsetToLineCol(f *testing.F) {
	f.Add([]byte("line1\nline2\nline3"), uint(0))
	f.Add([]byte(""), uint(5))
	f.Add([]byte("single line"), uint(50))

	f.Fuzz(func(t *testing.T, content []byte, rawOffset uint) {
		lines := BuildLineIndex(content)
		if len(lines) == 0 {
			t.Fatal("line index must always include the first line")
		}

		offset := 0
		if len(content) > 0 {
			offset = int(rawOffset % uint(len(content)))
		}

		line, col := OffsetToLineCol(lines, offset)
		if line < 1 || col < 1 {
			t.Fatalf("invalid position: line=%d col=%d", line, col)
		}

		if line > len(lines) {
			t.Fatalf("line out of bounds: line=%d lines=%d", line, len(lines))
		}

		start := lines[line-1]
		if start+col-1 > len(content) {
			t.Fatalf("column points past content: start=%d col=%d len=%d", start, col, len(content))
		}
	})
}
