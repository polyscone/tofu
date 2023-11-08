package smtp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/wneessen/go-mail"
)

var client = http.Client{Timeout: 10 * time.Second}

type ClientConfig struct {
	ResendAPIKey string
}

type ClientConfigReader interface {
	Read(ctx context.Context) (*ClientConfig, error)
}

type Client struct {
	config ClientConfigReader
	client *mail.Client
	logger *slog.Logger
}

func NewClient(logger *slog.Logger, config ClientConfigReader) (*Client, error) {
	client, err := mail.NewClient("localhost", mail.WithPort(25), mail.WithTLSPolicy(mail.TLSOpportunistic))
	if err != nil {
		return nil, err
	}

	c := &Client{
		config: config,
		client: client,
		logger: logger,
	}

	return c, nil
}

func (c *Client) sendDial(ctx context.Context, _msgs []Msg) error {
	msgs := make([]*mail.Msg, len(_msgs))
	for i, msg := range _msgs {
		m := mail.NewMsg()

		if msg.From != "" {
			if err := m.From(msg.From); err != nil {
				return fmt.Errorf("from address: %w", err)
			}
		}

		if msg.ReplyTo != "" {
			if err := m.ReplyTo(msg.ReplyTo); err != nil {
				return fmt.Errorf("reply-to address: %w", err)
			}
		}

		if len(msg.To) > 0 {
			if err := m.To(msg.To...); err != nil {
				return fmt.Errorf("to address: %w", err)
			}
		}

		if len(msg.Cc) > 0 {
			if err := m.Cc(msg.Cc...); err != nil {
				return fmt.Errorf("cc address: %w", err)
			}
		}

		if len(msg.Bcc) > 0 {
			if err := m.Bcc(msg.Bcc...); err != nil {
				return fmt.Errorf("bcc address: %w", err)
			}
		}

		m.Subject(msg.Subject)

		switch {
		case msg.Plain != "" && msg.HTML != "":
			m.SetBodyString(mail.TypeTextPlain, msg.Plain)
			m.AddAlternativeString(mail.TypeTextHTML, msg.HTML)

		case msg.Plain != "":
			m.SetBodyString(mail.TypeTextPlain, msg.Plain)

		case msg.HTML != "":
			m.SetBodyString(mail.TypeTextHTML, msg.HTML)
		}

		msgs[i] = m
	}

	if err := c.client.DialAndSendWithContext(ctx, msgs...); err != nil {
		return fmt.Errorf("dial and send: %w", err)
	}

	return nil
}

func (c *Client) sendResendAPI(ctx context.Context, msgs []Msg, apiKey string) error {
	for _, msg := range msgs {
		data := map[string]any{
			"from":    msg.From,
			"to":      msg.To,
			"subject": msg.Subject,
		}

		if msg.ReplyTo != "" {
			data["reply_to"] = msg.ReplyTo
		}

		if len(msg.Cc) > 0 {
			data["cc"] = msg.Cc
		}

		if len(msg.Bcc) > 0 {
			data["bcc"] = msg.Bcc
		}

		if msg.Plain != "" {
			data["text"] = msg.Plain
		}

		if msg.HTML != "" {
			data["html"] = msg.HTML
		}

		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal API request data: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, client.Timeout)
		defer cancel()

		endpoint := "https://api.resend.com/emails"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("new API request: %w", err)
		}

		req.Header.Set("content-type", "application/json")
		req.Header.Set("authorization", "Bearer "+apiKey)

		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("do API request: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode < 200 || res.StatusCode > 299 {
			b, err := io.ReadAll(res.Body)
			if err != nil {
				return fmt.Errorf("read API response body: %w", err)
			}

			var data struct {
				StatusCode int
				Name       string
				Message    string
			}
			if err := json.Unmarshal(b, &data); err != nil {
				return fmt.Errorf("unmarshal API response data: %w", err)
			}

			return fmt.Errorf("status code %v: %v: %v", data.StatusCode, data.Name, data.Message)
		}
	}

	return nil
}

func (c *Client) Send(ctx context.Context, msgs ...Msg) error {
	config, err := c.config.Read(ctx)
	if err != nil {
		return fmt.Errorf("resend API key: %w", err)
	}

	var attempts int
	var errs errsx.Slice

	if config.ResendAPIKey != "" {
		attempts++
		err := c.sendResendAPI(ctx, msgs, config.ResendAPIKey)
		if err == nil {
			return nil
		}

		errs.Append(fmt.Errorf("resend API: %w", err))
	}

	attempts++
	if err := c.sendDial(ctx, msgs); err != nil {
		errs.Append(fmt.Errorf("dial: %w", err))
	}

	if n := len(errs); n > 0 && n < attempts {
		if c.logger != nil {
			c.logger.Info("email sent by fallback", "error", errs)
		}

		return nil
	}

	if errs != nil {
		return errs
	}

	return nil
}
