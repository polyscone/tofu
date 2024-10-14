package smtp

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/errsx"
)

var crlfReplacer = strings.NewReplacer("\r", "", "\n", "")

type Attachment struct {
	filename string
	bytes    []byte
}

type Body struct {
	contentType string
	content     string
}

type Config struct {
	Auth                  smtp.Auth
	StartTLS              bool
	TLSInsecureSkipVerify bool
}

type Email struct {
	boundary    string
	from        string
	to          []string
	cc          []string
	bcc         []string
	replyTo     []string
	subject     string
	bodies      []Body
	attachments []Attachment
	mailer      Mailer
}

func NewEmail() (*Email, error) {
	buf := make([]byte, 30)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, fmt.Errorf("read random boundary bytes: %w", err)
	}

	boundary := fmt.Sprintf("%x", buf)
	if strings.ContainsAny(boundary, `()<>@,;:\"/[]?= `) {
		boundary = `"` + boundary + `"`
	}

	return &Email{boundary: boundary}, nil
}

func (e *Email) SetFrom(from string) error {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("parse address: %w", err)
	}

	addr.Name = ""

	from = strings.Trim(strings.ReplaceAll(addr.String(), "<@>", ""), "<>")
	if from == "" {
		return errors.New("cannot be empty")
	}

	e.from = from

	return nil
}

func (e *Email) addAddresses(_type string, addresses string) error {
	list, err := mail.ParseAddressList(addresses)
	if err != nil {
		return fmt.Errorf("parse address list: %w", err)
	}

	for _, addr := range list {
		addr.Name = ""

		address := strings.Trim(strings.ReplaceAll(addr.String(), "<@>", ""), "<>")
		if address == "" {
			return errors.New("cannot be empty")
		}

		switch _type {
		case "to":
			e.to = append(e.to, address)

		case "cc":
			e.cc = append(e.cc, address)

		case "bcc":
			e.bcc = append(e.bcc, address)

		case "replyTo":
			e.replyTo = append(e.replyTo, address)

		default:
			return fmt.Errorf("unknown address type %q", _type)
		}
	}

	return nil
}

func (e *Email) AddTo(addresses string) error {
	return e.addAddresses("to", addresses)
}

func (e *Email) AddCc(addresses string) error {
	return e.addAddresses("cc", addresses)
}

func (e *Email) AddBcc(addresses string) error {
	return e.addAddresses("bcc", addresses)
}

func (e *Email) AddReplyTo(addresses string) error {
	return e.addAddresses("replyTo", addresses)
}

func (e *Email) SetSubject(subject string) error {
	e.subject = crlfReplacer.Replace(subject)

	return nil
}

func (e *Email) AddBody(contentType, content string) error {
	if strings.TrimSpace(contentType) == "" {
		return errors.New("content type must be populated")
	}
	if strings.TrimSpace(content) == "" {
		return errors.New("content must be populated")
	}

	e.bodies = append(e.bodies, Body{
		contentType: contentType,
		content:     content,
	})

	return nil
}

func (e *Email) AddAttachment(filename string, r io.Reader) error {
	filename = crlfReplacer.Replace(filename)
	if filename == "" {
		return errors.New("filename cannot be empty")
	}
	if r == nil {
		return errors.New("attachment reader cannot be empty")
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read all: %w", err)
	}

	e.attachments = append(e.attachments, Attachment{
		filename: filename,
		bytes:    b,
	})

	return nil
}

func (e *Email) Send(addr string, config *Config) error {
	if e.from == "" {
		return errors.New("from address cannot be empty")
	}

	to := strings.Join(e.to, ", ")
	if to == "" {
		return errors.New("to address cannot be empty")
	}

	var msg bytes.Buffer

	msg.WriteString("Date: " + time.Now().UTC().Format(time.RFC822) + "\r\n")
	msg.WriteString("From: " + e.from + "\r\n")
	msg.WriteString("To: " + to + "\r\n")

	if cc := strings.Join(e.cc, ", "); cc != "" {
		msg.WriteString("Cc: " + cc + "\r\n")
	}

	// Skip adding Bcc to the message here and go straight to Reply-To because
	// Bcc addresses are supposed to be "blind" and not included
	// Later on they will be used as part of the send process, but they shouldn't
	// appear in the final message data

	if replyTo := strings.Join(e.replyTo, ", "); replyTo != "" {
		msg.WriteString("Reply-To: " + replyTo + "\r\n")
	}

	msg.WriteString("Subject: " + mimeEncode(e.subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")

	if len(e.attachments) > 0 {
		msg.WriteString(`Content-Type: multipart/mixed; boundary="` + e.boundary + "\"\r\n")
	} else {
		msg.WriteString(`Content-Type: multipart/alternative; boundary="` + e.boundary + "\"\r\n")
	}

	for _, body := range e.bodies {
		msg.WriteString("\r\n")
		msg.WriteString("--" + e.boundary + "\r\n")
		msg.WriteString("Content-Type: " + body.contentType + "; charset=utf-8\r\n")
		msg.WriteString("Content-Disposition: inline\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")

		writeBase64Lines(&msg, []byte(body.content))
	}

	for _, a := range e.attachments {
		msg.WriteString("\r\n")
		msg.WriteString("--" + e.boundary + "\r\n")
		msg.WriteString("Content-Type: application/octet-stream\r\n")
		msg.WriteString("Content-Disposition: attachment; filename=" + mimeEncode(a.filename) + "\r\n")
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString("\r\n")

		writeBase64Lines(&msg, a.bytes)
	}

	msg.WriteString("\r\n")
	msg.WriteString("--" + e.boundary + "--\r\n")

	if len(e.to) == 0 {
		return errors.New("to address cannot be empty")
	}

	if addr = strings.TrimSpace(addr); addr == "" {
		return errors.New("server address cannot be empty")
	}

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer c.Close()

	if config != nil {
		if config.StartTLS {
			if ok, _ := c.Extension("STARTTLS"); ok {
				host, _, _ := net.SplitHostPort(addr)
				tlsConfig := &tls.Config{
					ServerName:         host,
					InsecureSkipVerify: config.TLSInsecureSkipVerify,
				}
				if err := c.StartTLS(tlsConfig); err != nil {
					return fmt.Errorf("start TLS: %w", err)
				}
			}
		}

		if config.Auth != nil {
			if ok, _ := c.Extension("AUTH"); !ok {
				return errors.New("auth: server doesn't support AUTH")
			}

			if err := c.Auth(config.Auth); err != nil {
				return fmt.Errorf("auth: %w", err)
			}
		}
	}

	if err := c.Mail(e.from); err != nil {
		return fmt.Errorf("mail: %w", err)
	}

	var errs errsx.Slice
	for _, addr := range e.to {
		if err := c.Rcpt(addr); err != nil {
			errs.Append(fmt.Errorf("rcpt: to: %v: %w", addr, err))
		}
	}
	for _, addr := range e.cc {
		if err := c.Rcpt(addr); err != nil {
			errs.Append(fmt.Errorf("rcpt: cc: %v: %w", addr, err))
		}
	}
	for _, addr := range e.bcc {
		if err := c.Rcpt(addr); err != nil {
			errs.Append(fmt.Errorf("rcpt: bcc: %v: %w", addr, err))
		}
	}
	if err := errs.Err(); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data writer: %w", err)
	}

	if _, err := msg.WriteTo(w); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	if err := c.Quit(); err != nil {
		return fmt.Errorf("quit: %w", err)
	}

	return nil
}

func mimeEncode(s string) string {
	// Choose between B-Encoding and Q-Encoding based on the characters in the string
	// Certain characters aren't allowed in a message (see RFC 2047 section 5.3)
	// If the string contains any characters that aren't allowed in the Q-Encoding
	// then we use the B-Encoding instead
	if strings.ContainsAny(s, "\"#$%&'(),.:;<>@[]^`{|}~") {
		s = mime.BEncoding.Encode("utf-8", s)
	} else {
		s = mime.QEncoding.Encode("utf-8", s)
	}

	// For whatever reason the Go standard library's mime.[BQ]Encoding functions
	// return the encoded word with space separators rather than a CRLF newline, so
	// we need to replace those spaces with \r\n ourselves
	return strings.ReplaceAll(s, "?= =?", "?=\r\n=?")
}

func writeBase64Lines(dst io.ByteWriter, src []byte) {
	b := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(b, src)

	for i, l := 0, len(b); i < l; i++ {
		dst.WriteByte(b[i])

		if (i+1)%76 == 0 {
			dst.WriteByte('\r')
			dst.WriteByte('\n')
		}
	}

	dst.WriteByte('\r')
	dst.WriteByte('\n')
}
