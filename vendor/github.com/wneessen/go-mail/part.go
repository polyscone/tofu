// SPDX-FileCopyrightText: 2022-2023 The go-mail Authors
//
// SPDX-License-Identifier: MIT

package mail

import (
	"bytes"
	"io"
)

// PartOption returns a function that can be used for grouping Part options
type PartOption func(*Part)

// Part is a part of the Msg
type Part struct {
	ctype ContentType
	cset  Charset
	desc  string
	enc   Encoding
	del   bool
	w     func(io.Writer) (int64, error)
}

// GetContent executes the WriteFunc of the Part and returns the content as byte slice
func (p *Part) GetContent() ([]byte, error) {
	var b bytes.Buffer
	if _, err := p.w(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// GetCharset returns the currently set Charset of the Part
func (p *Part) GetCharset() Charset {
	return p.cset
}

// GetContentType returns the currently set ContentType of the Part
func (p *Part) GetContentType() ContentType {
	return p.ctype
}

// GetEncoding returns the currently set Encoding of the Part
func (p *Part) GetEncoding() Encoding {
	return p.enc
}

// GetWriteFunc returns the currently set WriterFunc of the Part
func (p *Part) GetWriteFunc() func(io.Writer) (int64, error) {
	return p.w
}

// GetDescription returns the currently set Content-Description of the Part
func (p *Part) GetDescription() string {
	return p.desc
}

// SetContent overrides the content of the Part with the given string
func (p *Part) SetContent(c string) {
	buf := bytes.NewBufferString(c)
	p.w = writeFuncFromBuffer(buf)
}

// SetContentType overrides the ContentType of the Part
func (p *Part) SetContentType(c ContentType) {
	p.ctype = c
}

// SetCharset overrides the Charset of the Part
func (p *Part) SetCharset(c Charset) {
	p.cset = c
}

// SetEncoding creates a new mime.WordEncoder based on the encoding setting of the message
func (p *Part) SetEncoding(e Encoding) {
	p.enc = e
}

// SetDescription overrides the Content-Description of the Part
func (p *Part) SetDescription(d string) {
	p.desc = d
}

// SetWriteFunc overrides the WriteFunc of the Part
func (p *Part) SetWriteFunc(w func(io.Writer) (int64, error)) {
	p.w = w
}

// Delete removes the current part from the parts list of the Msg by setting the
// del flag to true. The msgWriter will skip it then
func (p *Part) Delete() {
	p.del = true
}

// WithPartCharset overrides the default Part charset
func WithPartCharset(c Charset) PartOption {
	return func(p *Part) {
		p.cset = c
	}
}

// WithPartEncoding overrides the default Part encoding
func WithPartEncoding(e Encoding) PartOption {
	return func(p *Part) {
		p.enc = e
	}
}

// WithPartContentDescription overrides the default Part Content-Description
func WithPartContentDescription(d string) PartOption {
	return func(p *Part) {
		p.desc = d
	}
}
