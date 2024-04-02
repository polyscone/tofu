// SPDX-FileCopyrightText: 2022-2023 The go-mail Authors
//
// SPDX-License-Identifier: MIT

package mail

import (
	"fmt"
	"io"
)

// ErrNoOutWriter is an error message that should be used if a Base64LineBreaker has no out io.Writer set
const ErrNoOutWriter = "no io.Writer set for Base64LineBreaker"

// Base64LineBreaker is a io.WriteCloser that writes Base64 encoded data streams
// with line breaks at a given line length
type Base64LineBreaker struct {
	line [MaxBodyLength]byte
	used int
	out  io.Writer
}

var nl = []byte(SingleNewLine)

// Write writes the data stream and inserts a SingleNewLine when the maximum
// line length is reached
func (l *Base64LineBreaker) Write(b []byte) (n int, err error) {
	if l.out == nil {
		err = fmt.Errorf(ErrNoOutWriter)
		return
	}
	if l.used+len(b) < MaxBodyLength {
		copy(l.line[l.used:], b)
		l.used += len(b)
		return len(b), nil
	}

	n, err = l.out.Write(l.line[0:l.used])
	if err != nil {
		return
	}
	excess := MaxBodyLength - l.used
	l.used = 0

	n, err = l.out.Write(b[0:excess])
	if err != nil {
		return
	}

	n, err = l.out.Write(nl)
	if err != nil {
		return
	}

	return l.Write(b[excess:])
}

// Close closes the Base64LineBreaker and writes any access data that is still
// unwritten in memory
func (l *Base64LineBreaker) Close() (err error) {
	if l.used > 0 {
		_, err = l.out.Write(l.line[0:l.used])
		if err != nil {
			return
		}
		_, err = l.out.Write(nl)
	}

	return
}
