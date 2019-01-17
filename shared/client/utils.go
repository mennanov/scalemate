package client

import (
	"fmt"
	"io"
	"os"
)

// DisplayMessage writes a msg to the provided io.Writer and handles an error by calling itself.
func DisplayMessage(w io.Writer, msg string) {
	if _, err := fmt.Fprintln(w, msg); err != nil {
		DisplayMessage(os.Stderr, err.Error())
	}
}

// DisplayMessagef is similar to DisplayMessage, but with formatting.
func DisplayMessagef(w io.Writer, format string, a ...interface{}) {
	if _, err := fmt.Fprintf(w, format, a); err != nil {
		DisplayMessage(os.Stderr, err.Error())
	}
}
