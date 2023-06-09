package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

const codeInvalidToNumber = 21211

// Client represents the data required to interact with the Twilio API.
type Client struct {
	client   *http.Client
	sid      string
	token    string
	Endpoint string
}

// NewTwilioClient will return a newly instantiated Client.
func NewTwilioClient(client *http.Client, sid, token string) *Client {
	return &Client{
		client: client,
		sid:    sid,
		token:  token,
	}
}

func (c *Client) isValid() error {
	if strings.TrimSpace(c.sid) == "" {
		return errors.Tracef("sid must be populated")
	}
	if want := "AC"; !strings.HasPrefix(c.sid, want) {
		return errors.Tracef("sid must be prefixed with the string %q", want)
	}
	if want := 34; len(c.sid) != want {
		return errors.Tracef("sid must be %d characters in length", want)
	}
	if strings.TrimSpace(c.token) == "" {
		return errors.Tracef("token must be populated")
	}
	return nil
}

// Send will use the Twilio API to send an SMS message using the given data.
func (c *Client) Send(ctx context.Context, from, to, body string) error {
	if err := c.isValid(); err != nil {
		return errors.Tracef(err)
	}

	from = strings.TrimSpace(from)
	if from == "" {
		return errors.Tracef("from must be populated")
	}
	if !strings.HasPrefix(from, "+") {
		return errors.Tracef("from must be prefixed with a +")
	}

	to = strings.TrimSpace(to)
	if to == "" {
		return errors.Tracef("to must be populated")
	}
	if !strings.HasPrefix(to, "+") {
		return errors.Tracef("to must be prefixed with a +")
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return errors.Tracef("body must be populated")
	}
	if maxLen := 1600; len(body) > maxLen {
		return errors.Tracef("body is too long, want max length %d, got %d", maxLen, len(body))
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.twilio.com"
	}
	endpoint += "/2010-04-01/Accounts/" + c.sid + "/Messages.json"

	data := url.Values{
		"To":   {to},
		"From": {from},
		"Body": {body},
	}

	ctx, cancel := context.WithTimeout(ctx, c.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Tracef(err)
	}

	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.sid, c.token)

	res, err := c.client.Do(req)
	if err != nil {
		return errors.Tracef(err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return errors.Tracef(err)
		}

		var data struct {
			Code    int
			Message string
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return errors.Tracef(err)
		}

		switch data.Code {
		case codeInvalidToNumber:
			if from == to {
				return errors.Tracef(ErrInvalidNumber, "the from and to numbers cannot be the same")
			}

			return errors.Tracef(ErrInvalidNumber, "%v is an invalid number", to)

		default:
			return errors.Tracef("error %v: %v", data.Code, data.Message)
		}
	}

	return nil
}
