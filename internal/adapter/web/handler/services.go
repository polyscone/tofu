package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/session"
)

type CSRF struct {
	ctx context.Context
}

func (c CSRF) Token() string {
	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(c.ctx))
}

type emailContent struct {
	Subject string
	Plain   string
	HTML    string
}

type EmailRecipients struct {
	From    string
	ReplyTo string
	To      []string
	Cc      []string
	Bcc     []string
}

type Vars map[string]any

func (v Vars) Merge(rhs Vars) Vars {
	if v == nil {
		v = make(Vars, len(rhs))
	}

	for key, value := range rhs {
		v[key] = value
	}

	return v
}

type AddrData struct {
	Scheme   string
	Host     string
	Hostname string
	Port     string
}

type AppData struct {
	Name        string
	Description string
}

type SessionData struct {
	// General session keys
	Flash    string
	Redirect string

	// Account session keys
	UserID                   int
	Email                    string
	HasVerifiedTOTP          bool
	TOTPMethod               string
	IsAwaitingTOTP           bool
	IsAuthenticated          bool
	PasswordKnownBreachCount int
}

type ViewData struct {
	View         string
	Status       int
	CSRF         CSRF
	ErrorMessage string
	Errors       errors.Map
	Form         url.Values
	Query        url.Values
	Addr         AddrData
	App          AppData
	Session      SessionData
	Vars         Vars
}

func (v ViewData) WithBook(book any) ViewData {
	v.Vars["__Book"] = book

	return v
}

type ViewDataFunc func(data *ViewData)

type emailData struct {
	Addr AddrData
	App  AppData
	Vars Vars
}

type emailDataFunc func(data *emailData)

type Services struct {
	*Tenant
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
	funcs       template.FuncMap
	viewVars    map[string]Vars
	mux         *router.ServeMux
	Sessions    *session.Manager
}

func NewServices(mux *router.ServeMux, tenant *Tenant, files fs.FS) *Services {
	sessions := session.NewManager(tenant.Repo.Web)
	funcs := template.FuncMap{
		"HTML":       func(s string) template.HTML { return template.HTML(s) },
		"HTMLAttr":   func(s string) template.HTMLAttr { return template.HTMLAttr(s) },
		"URL":        func(s string) template.URL { return template.URL(s) },
		"StatusText": http.StatusText,
		"Path":       mux.Path,
		"Add":        func(a, b int) int { return a + b },
		"Sub":        func(a, b int) int { return a - b },
		"Mul":        func(a, b int) int { return a * b },
		"Div":        func(a, b int) int { return a / b },
		"Mod":        func(a, b int) int { return a % b },
		"Ints": func(start, end int) []int {
			n := end - start
			ints := make([]int, n)
			for i := 0; i < n; i++ {
				ints[i] = start + i
			}

			return ints
		},
		"QueryReplace": func(q url.Values, pairs ...any) (string, error) {
			if len(pairs)%2 == 1 {
				return "", errors.Tracef("QueryReplace expects pairs of key value replacements")
			}

			u, err := url.Parse("?" + q.Encode())
			if err != nil {
				return "", errors.Tracef(err)
			}

			qq := u.Query()
			for i := 0; i < len(pairs); i += 2 {
				key := fmt.Sprintf("%v", pairs[i])
				value := pairs[i+1]

				if value == nil {
					qq.Del(key)

					continue
				}

				qq.Set(key, fmt.Sprintf("%v", value))
			}

			return qq.Encode(), nil
		},
		"FormatTime": func(t time.Time, format string) string {
			switch format {
			case "RFC3339":
				return t.Format(time.RFC3339)
			}

			return t.Format(format)
		},
	}

	return &Services{
		Tenant:    tenant,
		files:     files,
		templates: make(map[string]*template.Template),
		funcs:     funcs,
		viewVars:  make(map[string]Vars),
		mux:       mux,
		Sessions:  sessions,
	}
}

func (svc *Services) RenewSession(ctx context.Context) ([]byte, error) {
	if err := csrf.RenewToken(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	if err := svc.Sessions.Renew(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	return csrf.MaskedToken(ctx), nil
}

func (svc *Services) emptyPassport(ctx context.Context) passport.Passport {
	return passport.New(ctx, svc.Sessions, &account.User{})
}

func (svc *Services) Passport(ctx context.Context) passport.Passport {
	if svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		return svc.emptyPassport(ctx)
	}

	userID := svc.Sessions.GetInt(ctx, sess.UserID)
	user, err := svc.Repo.Account.FindUserByID(ctx, userID)
	if err != nil {
		return svc.emptyPassport(ctx)
	}

	return passport.New(ctx, svc.Sessions, user)
}

func (svc *Services) PassportByEmail(ctx context.Context, email string) (passport.Passport, error) {
	user, err := svc.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return svc.emptyPassport(ctx), errors.Tracef(err)
	}

	return passport.New(ctx, svc.Sessions, user), nil
}

func (svc *Services) Path(name string, paramArgPairs ...string) string {
	return svc.mux.Path(name, paramArgPairs...)
}

func (svc *Services) SetViewVars(name string, vars Vars) {
	if _, ok := svc.viewVars[name]; ok {
		panic(fmt.Sprintf("default view vars already set for %q", name))
	}

	svc.viewVars[name] = vars
}

func (svc *Services) template(name string, patterns ...string) *template.Template {
	svc.templatesMu.RLock()

	if tmpl := svc.templates[name]; tmpl != nil && !svc.Tenant.Dev {
		svc.templatesMu.RUnlock()

		return tmpl
	}

	svc.templatesMu.RUnlock()

	svc.templatesMu.Lock()
	defer svc.templatesMu.Unlock()

	tmpl := template.New(name).Option("missingkey=default").Funcs(svc.funcs)

	for _, pattern := range patterns {
		tmpl = errors.Must(tmpl.ParseFS(svc.files, pattern))
	}

	svc.templates[name] = tmpl

	return tmpl
}

func (svc *Services) email(name string) *template.Template {
	return svc.template(name, "email/"+name+".tmpl")
}

func (svc *Services) emailContentFunc(name string, dataFunc emailDataFunc) (emailContent, error) {
	data := emailData{
		Addr: AddrData{
			Scheme:   svc.Tenant.Scheme,
			Host:     svc.Tenant.Host,
			Hostname: svc.Tenant.Hostname,
			Port:     svc.Tenant.Port,
		},
		App: AppData{
			Name:        app.Name,
			Description: app.Description,
		},
	}

	if vars, ok := svc.viewVars[name]; ok {
		data.Vars = data.Vars.Merge(vars)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	email := svc.email(name)

	var content emailContent
	var buf bytes.Buffer

	for _, name := range []string{"subject", "text", "html"} {
		tmpl := email.Lookup(name)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if err := tmpl.Execute(&buf, data); err != nil {
			return content, errors.Tracef(err)
		}

		switch name {
		case "subject":
			content.Subject = buf.String()

		case "text":
			content.Plain = buf.String()

		case "html":
			content.HTML = buf.String()
		}
	}

	return content, nil
}

func (svc *Services) emailContent(name string, vars Vars) (emailContent, error) {
	return svc.emailContentFunc(name, func(data *emailData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (svc *Services) SendEmail(ctx context.Context, recipients EmailRecipients, name string, vars Vars) error {
	content, err := svc.emailContent(name, vars)
	if err != nil {
		return errors.Tracef(err)
	}

	msg := smtp.Msg{
		From:    recipients.From,
		ReplyTo: recipients.ReplyTo,
		To:      recipients.To,
		Cc:      recipients.Cc,
		Bcc:     recipients.Bcc,
		Subject: content.Subject,
		Plain:   content.Plain,
		HTML:    content.HTML,
	}

	return errors.Tracef(svc.Tenant.Email.Send(ctx, msg))
}

func (svc *Services) SendSMS(ctx context.Context, to, body string) error {
	return errors.Tracef(svc.Tenant.SMS.Send(ctx, svc.Tenant.SMSFrom, to, body))
}

func (svc *Services) SendTOTPSMS(email string) error {
	ctx := context.Background()

	user, err := svc.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return errors.Tracef(err)
	}

	totp, err := user.GenerateTOTP()
	if err != nil {
		return errors.Tracef(err)
	}

	err = svc.SendSMS(ctx, user.TOTPTelephone, totp)

	return errors.Tracef(err)
}

func (svc *Services) view(name string) *template.Template {
	return svc.template(name, "partial/*.tmpl", "view/"+name+".tmpl", "master.tmpl")
}

func (svc *Services) ViewFunc(w http.ResponseWriter, r *http.Request, status int, name string, dataFunc ViewDataFunc) {
	ctx := r.Context()

	data := ViewData{
		View:   name,
		Status: status,
		CSRF:   CSRF{ctx: ctx},
		Form:   r.PostForm,
		Query:  r.URL.Query(),
		Addr: AddrData{
			Scheme:   svc.Tenant.Scheme,
			Host:     svc.Tenant.Host,
			Hostname: svc.Tenant.Hostname,
			Port:     svc.Tenant.Port,
		},
		App: AppData{
			Name:        app.Name,
			Description: app.Description,
		},
		Session: SessionData{
			// Global session keys
			Flash:    svc.Sessions.PopString(ctx, sess.Flash),
			Redirect: svc.Sessions.GetString(ctx, sess.Redirect),

			// Account session keys
			UserID:                   svc.Sessions.GetInt(ctx, sess.UserID),
			Email:                    svc.Sessions.GetString(ctx, sess.Email),
			HasVerifiedTOTP:          svc.Sessions.GetBool(ctx, sess.HasVerifiedTOTP),
			TOTPMethod:               svc.Sessions.GetString(ctx, sess.TOTPMethod),
			IsAwaitingTOTP:           svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
			IsAuthenticated:          svc.Sessions.GetBool(ctx, sess.IsAuthenticated),
			PasswordKnownBreachCount: svc.Sessions.GetInt(ctx, sess.PasswordKnownBreachCount),
		},
	}

	if vars, ok := svc.viewVars[name]; ok {
		data.Vars = data.Vars.Merge(vars)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	// Make sure the current view name isn't overwritten by a user function
	data.View = name

	var buf bytes.Buffer
	if err := svc.view(name).ExecuteTemplate(&buf, "master", data); err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		httputil.LogError(r, errors.Tracef(err))
	}
}

func (svc *Services) View(w http.ResponseWriter, r *http.Request, status int, name string, vars Vars) {
	svc.ViewFunc(w, r, status, name, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (svc *Services) ErrorViewFunc(w http.ResponseWriter, r *http.Request, err error, name string, dataFunc ViewDataFunc) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, errors.Tracef(err))

	status := httputil.ErrorStatus(err)

	svc.ViewFunc(w, r, status, name, func(data *ViewData) {
		switch {
		case errors.Is(err, http.ErrHandlerTimeout):
			data.ErrorMessage = "the server took too long to respond"

		case errors.Is(err, app.ErrMalformedInput),
			errors.Is(err, app.ErrInvalidInput):

			data.ErrorMessage = "invalid input"

			if trace, ok := err.(errors.Trace); ok {
				data.Errors = trace.Fields()
			}

		case errors.Is(err, csrf.ErrEmptyToken):
			data.ErrorMessage = "empty CSRF token"

		case errors.Is(err, csrf.ErrInvalidToken):
			data.ErrorMessage = "invalid CSRF token"

		default:
			data.ErrorMessage = "an error has occurred"
		}

		if dataFunc != nil {
			dataFunc(data)
		}
	})

	return true
}

func (svc *Services) ErrorView(w http.ResponseWriter, r *http.Request, err error, name string, vars Vars) bool {
	return svc.ErrorViewFunc(w, r, errors.Tracef(err), name, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (svc *Services) ErrorJSON(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, errors.Tracef(err))

	var displayOK bool
	status := httputil.ErrorStatus(err)

	switch {
	case errors.Is(err, httputil.ErrBadJSON):
		status, displayOK = http.StatusBadRequest, true

	case errors.Is(err, httputil.ErrExpectedJSON):
		status, displayOK = http.StatusUnsupportedMediaType, true

	default:
		switch {
		case errors.Is(err, http.ErrHandlerTimeout),
			errors.Is(err, app.ErrMalformedInput),
			errors.Is(err, app.ErrInvalidInput),
			errors.Is(err, app.ErrUnauthorised),
			errors.Is(err, csrf.ErrEmptyToken),
			errors.Is(err, csrf.ErrInvalidToken):

			displayOK = true
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if displayOK && 400 <= status && status <= 499 {
		detail["error"] = err.Error()

		if trace, ok := err.(errors.Trace); ok {
			fields := trace.Fields()

			if fields != nil {
				detail["fields"] = fields
			}
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		httputil.LogError(r, errors.Tracef(err))
	}

	return true
}

func (svc *Services) JSON(w http.ResponseWriter, r *http.Request, data any) bool {
	w.Header().Set("content-type", "application/json")

	return !svc.ErrorJSON(w, r, errors.Tracef(json.NewEncoder(w).Encode(data)))
}

func (svc *Services) Pagination(r *http.Request) (int, int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}
	if page < 1 {
		page = 1
	}

	const maxSize = 100

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = 10
	}
	if size < 1 {
		size = 1
	}
	if size > maxSize {
		size = maxSize
	}

	return page, size
}
