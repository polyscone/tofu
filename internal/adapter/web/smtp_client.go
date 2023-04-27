package web

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/wneessen/go-mail"
)

type MailClient struct {
	client *mail.Client
}

func NewMailClient(host string, port int) (*MailClient, error) {
	c, err := mail.NewClient(host, mail.WithPort(port), mail.WithTLSPolicy(mail.TLSOpportunistic))
	if err != nil {
		return nil, errors.Tracef(err)
	}

	return &MailClient{client: c}, nil
}

func (m *MailClient) Send(ctx context.Context, _msgs ...smtp.Msg) error {
	msgs := make([]*mail.Msg, len(_msgs))
	for i, msg := range _msgs {
		m := mail.NewMsg()

		if msg.From != "" {
			if err := m.From(msg.From); err != nil {
				return errors.Tracef(err)
			}
		}

		if msg.ReplyTo != "" {
			if err := m.ReplyTo(msg.ReplyTo); err != nil {
				return errors.Tracef(err)
			}
		}

		if len(msg.To) != 0 {
			if err := m.To(msg.To...); err != nil {
				return errors.Tracef(err)
			}
		}

		if len(msg.Cc) != 0 {
			if err := m.Cc(msg.Cc...); err != nil {
				return errors.Tracef(err)
			}
		}

		if len(msg.Bcc) != 0 {
			if err := m.Bcc(msg.Bcc...); err != nil {
				return errors.Tracef(err)
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

	return errors.Tracef(m.client.DialAndSendWithContext(ctx, msgs...))
}
