package cmd

import (
	"fmt"
	"io"
)

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

func writeString(w io.Writer, value string) {
	_, _ = io.WriteString(w, value)
}
