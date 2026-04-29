package main

import (
	"fmt"
	"os"
	"time"
)

// labelWidth is the fixed column width for step labels.
const labelWidth = 10

// noColor is true when ANSI escape codes should be suppressed  (non-TTY or
// NO_COLOR env is set).
var noColor bool

func init() {
	stat, err := os.Stderr.Stat()
	noColor = err != nil ||
		(stat.Mode()&os.ModeCharDevice) == 0 ||
		os.Getenv("NO_COLOR") != ""
}

// esc returns an ANSI escape sequence, or an empty string when colors are off.
func esc(code string) string {
	if noColor {
		return ""
	}
	return "\033[" + code + "m"
}

// logStep prints a labelled step line:
//
//	label       value
func logStep(label, value string) {
	fmt.Fprintf(os.Stderr, "  %s%-*s%s  %s\n",
		esc("2"), labelWidth, label, esc("0"), value)
}

// logWarn prints a yellow warning line.
func logWarn(value string) {
	fmt.Fprintf(os.Stderr, "  %s%-*s%s  %s%s%s\n",
		esc("33"), labelWidth, "warning", esc("0"),
		esc("33"), value, esc("0"))
}

// logChange prints a dim separator line used by the dev server to announce a
// file-change event.
func logChange(msg string) {
	fmt.Fprintf(os.Stderr, "\n  %s· %s%s\n", esc("2"), msg, esc("0"))
}

// logBlank prints a single blank line.
func logBlank() { fmt.Fprintln(os.Stderr) }

// logDone prints the final green "done" line with elapsed time.
func logDone(d time.Duration) {
	fmt.Fprintf(os.Stderr, "\n  %s%-*s%s  %s%s%s\n",
		esc("32;1"), labelWidth, "done", esc("0"),
		esc("2"), fmtDur(d), esc("0"))
}

// fatalf prints a red error label + message and exits with code 1.
func fatalf(label, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "\n  %s%-*s%s  %s\n",
		esc("31;1"), labelWidth, label, esc("0"), msg)
	os.Exit(1)
}

// ── formatting helpers ────────────────────────────────────────────────────────

// fmtSize formats a byte count as a human-readable size string.
func fmtSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.2f MB", float64(n)/1024/1024)
	}
}

// fmtSizeRatio formats two byte counts as "X.XX MB → Y.YY MB  (Z%)" where Z
// is the percentage of out relative to in.
func fmtSizeRatio(in, out int) string {
	if in == 0 {
		return fmtSize(out)
	}
	pct := float64(out) / float64(in) * 100
	return fmt.Sprintf("%s → %s  (%.1f%%)", fmtSize(in), fmtSize(out), pct)
}

// fmtDur formats a duration as "Xms" or "X.XXs".
func fmtDur(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
