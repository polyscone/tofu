package handler

import (
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/session"
)

type Tenant struct {
	Scheme   string
	Host     string
	Hostname string
	Port     string
	Dev      bool
	Insecure bool
	Proxies  []string
	Bus      command.Bus
	Broker   event.Broker
	Sessions session.Repo
	Tokens   token.Repo
	Mailer   smtp.Mailer
}
