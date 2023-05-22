package handler

import (
	"github.com/polyscone/tofu/internal/adapter/web/query"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/sms"
)

type Account struct {
	Users query.AccountUserRepo
}

type Web struct {
	Sessions session.Repo
	Tokens   query.TokenRepo
}

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
	Email    smtp.Mailer
	SMS      sms.Messager
	SMSFrom  string

	Account Account
	Web     Web
}
