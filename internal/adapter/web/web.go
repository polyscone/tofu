package web

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/internal/repo/sqlite"
)

var (
	NewSQLiteSessionRepo = sqlite.NewSessionRepo
	NewSQLiteTokenRepo   = sqlite.NewTokenRepo
)

type Mailer interface {
	Send(ctx context.Context, msgs ...Msg) error
}

type Msg struct {
	From    string
	ReplyTo string
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Plain   string
	HTML    string
}
