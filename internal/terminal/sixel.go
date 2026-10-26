package terminal

import (
	"bytes"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

// daPrefix starts a Primary Device Attributes reply: ESC [ ? <params> c.
var daPrefix = []byte("\x1b[?") //nolint:gochecknoglobals

// SixelSupported reports whether the attached terminal advertises Sixel graphics.
// It sends a Primary Device Attributes query (DA1) and looks for capability 4 in
// the reply. It returns false when stdin or stdout is not a terminal, or when no
// reply arrives within timeout.
//
// The reply is read on a goroutine. If the timeout expires, that goroutine may
// remain blocked on stdin until the process exits; this is acceptable for the
// short-lived CLI invocations that call this function.
func SixelSupported(timeout time.Duration) bool {
	stdin := int(os.Stdin.Fd())
	if !StdoutIsTerminal() || !term.IsTerminal(stdin) {
		return false
	}

	state, err := term.MakeRaw(stdin)
	if err != nil {
		return false
	}
	defer func() { _ = term.Restore(stdin, state) }()

	if _, err := os.Stdout.WriteString("\x1b[c"); err != nil {
		return false
	}

	replies := make(chan []byte, 1)
	go func() { replies <- readDAReply(os.Stdin) }()

	select {
	case reply := <-replies:
		return daReplyHasSixel(reply)
	case <-time.After(timeout):
		return false
	}
}

// readDAReply reads bytes until a 'c' terminator arrives, the reader fails, or
// the reply grows past a sane limit.
func readDAReply(r io.Reader) []byte {
	const maxReply = 256
	var buf []byte
	one := make([]byte, 1)
	for len(buf) < maxReply {
		n, err := r.Read(one)
		if n == 1 {
			buf = append(buf, one[0])
			if one[0] == 'c' && bytes.Contains(buf, daPrefix) {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return buf
}

// daReplyHasSixel reports whether a DA1 reply lists capability 4 (Sixel graphics).
func daReplyHasSixel(reply []byte) bool {
	_, rest, found := bytes.Cut(reply, daPrefix)
	if !found {
		return false
	}
	params, _, found := bytes.Cut(rest, []byte{'c'})
	if !found {
		return false
	}
	return slices.Contains(strings.Split(string(params), ";"), "4")
}
