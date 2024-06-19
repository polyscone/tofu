package smtp

import "context"

type Mailer interface {
	Send(ctx context.Context, msgs ...Msg) error
}

type Msg struct {
	From    string
	ReplyTo []string
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Plain   string
	HTML    string
}
