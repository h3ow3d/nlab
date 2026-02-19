// Package log provides shared logging helpers matching the nlab script conventions.
package log

import (
	"fmt"
	"os"
)

func Info(msg string)  { fmt.Printf("[+] %s\n", msg) }
func Ok(msg string)    { fmt.Printf("[✓] %s\n", msg) }
func Skip(msg string)  { fmt.Printf("[=] %s\n", msg) }
func Error(msg string) { fmt.Fprintf(os.Stderr, "[!] %s\n", msg) }
