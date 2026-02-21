// Package lab contains the core logic for the nlab CLI.
package lab

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ANSI escape codes.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
)

// colorize wraps msg in an ANSI color sequence only when stdout is a TTY.
func colorize(color, msg string) string {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return color + bold + msg + reset
	}
	return msg
}

// colorizeStderr wraps msg in color when stderr is a TTY.
func colorizeStderr(color, msg string) string {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return color + bold + msg + reset
	}
	return msg
}

func Info(msg string)  { fmt.Printf("%s %s\n", colorize(cyan, "[+]"), msg) }
func Ok(msg string)    { fmt.Printf("%s %s\n", colorize(green, "[✓]"), msg) }
func Skip(msg string)  { fmt.Printf("%s %s\n", colorize(yellow, "[=]"), msg) }
func Error(msg string) { fmt.Fprintf(os.Stderr, "%s %s\n", colorizeStderr(red, "[!]"), msg) }
