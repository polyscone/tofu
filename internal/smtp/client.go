package smtp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/errsx"
)

var client = http.Client{Timeout: 10 * time.Second}

type ClientConfig struct {
	EnvelopeEmail string
	ResendAPIKey  string
}

type ClientConfigReader interface {
	Read(ctx context.Context) (*ClientConfig, error)
}

type ResendMsg struct {
	msg    Msg
	apiKey string
	wg     *sync.WaitGroup
	errs   chan error
}

type Client struct {
	config ClientConfigReader
	resend chan ResendMsg
	logger *slog.Logger
}

func NewClient(logger *slog.Logger, config ClientConfigReader) (*Client, error) {
	c := &Client{
		config: config,
		resend: make(chan ResendMsg, 100),
		logger: logger,
	}

	go c.processResendAPIQueue()

	return c, nil
}

func (c *Client) send(ctx context.Context, msgs []Msg, envelopeEmail string) error {
	var errs errsx.Slice
SendLoop:
	for _, msg := range msgs {
		email, err := NewEmail()
		if err != nil {
			errs.Append(fmt.Errorf("from address: %w", err))

			continue SendLoop
		}

		if err := email.SetFrom(msg.From); err != nil {
			errs.Append(fmt.Errorf("from address: %w", err))

			continue SendLoop
		}

		for _, addr := range msg.ReplyTo {
			if err := email.AddReplyTo(addr); err != nil {
				errs.Append(fmt.Errorf("reply-to address: %w", err))

				continue SendLoop
			}
		}

		for _, addr := range msg.To {
			if err := email.AddTo(addr); err != nil {
				errs.Append(fmt.Errorf("to address: %w", err))

				continue SendLoop
			}
		}

		for _, addr := range msg.Cc {
			if err := email.AddCc(addr); err != nil {
				errs.Append(fmt.Errorf("cc address: %w", err))

				continue SendLoop
			}
		}

		for _, addr := range msg.Bcc {
			if err := email.AddBcc(addr); err != nil {
				errs.Append(fmt.Errorf("bcc address: %w", err))

				continue SendLoop
			}
		}

		if err := email.SetSubject(msg.Subject); err != nil {
			errs.Append(fmt.Errorf("subject: %w", err))

			continue SendLoop
		}

		if msg.Plain != "" {
			if err := email.AddBody("text/plain", msg.Plain); err != nil {
				errs.Append(fmt.Errorf("add plain body: %w", err))

				continue SendLoop
			}
		}

		if msg.HTML != "" {
			if err := email.AddBody("text/html", msg.HTML); err != nil {
				errs.Append(fmt.Errorf("add HTML body: %w", err))

				continue SendLoop
			}
		}

		config := Config{EnvelopeEmail: envelopeEmail}
		if err := email.Send("localhost:25", &config); err != nil {
			errs.Append(fmt.Errorf("send: %w", err))
		}
	}

	return errs.Err()
}

func (c *Client) sendResendAPI(ctx context.Context, msg Msg, apiKey string) error {
	data := map[string]any{
		"from":    msg.From,
		"to":      msg.To,
		"subject": msg.Subject,
	}

	if len(msg.ReplyTo) > 0 {
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

	return nil
}

func (c *Client) processResendAPIQueue() {
	// Resend rate limits at 2 req/s
	//
	// Throttling is per client and not per API key because we assume a
	// stable single API key for the majority of the time
	throttle := time.NewTicker(time.Second / 2)
	defer throttle.Stop()

	for resendMsg := range c.resend {
		<-throttle.C

		background.Go(func() {
			resendMsg.errs <- c.sendResendAPI(context.Background(), resendMsg.msg, resendMsg.apiKey)

			resendMsg.wg.Done()
		})
	}
}

func (c *Client) enqueueResendAPI(ctx context.Context, msgs []Msg, apiKey string) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(msgs))
	for _, msg := range msgs {
		wg.Add(1)

		c.resend <- ResendMsg{
			msg:    msg,
			apiKey: apiKey,
			wg:     &wg,
			errs:   errs,
		}
	}

	background.Go(func() {
		wg.Wait()

		close(errs)
	})

	var joined errsx.Slice
	for err := range errs {
		joined.Append(err)
	}

	return joined.Err()
}

func (c *Client) SendEmail(ctx context.Context, msgs ...Msg) error {
	config, err := c.config.Read(ctx)
	if err != nil {
		return fmt.Errorf("resend API key: %w", err)
	}

	var attempts int
	var errs errsx.Slice

	if config.ResendAPIKey != "" {
		attempts++
		err := c.enqueueResendAPI(ctx, msgs, config.ResendAPIKey)
		if err == nil {
			return nil
		}

		errs.Append(fmt.Errorf("resend API: %w", err))
	}

	attempts++
	if err := c.send(ctx, msgs, config.EnvelopeEmail); err != nil {
		errs.Append(fmt.Errorf("email message: %w", err))
	}

	// If one of the services produced an error but the emails were
	// sent anyway by a fallback then we want to return nil, but log
	// that an error occurred
	if n := len(errs); n > 0 && n < attempts {
		if c.logger != nil {
			c.logger.Info("email sent by fallback", "error", errs)
		}

		return nil
	}

	return errs.Err()
}
