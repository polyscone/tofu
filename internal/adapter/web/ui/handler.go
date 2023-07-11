package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/pkg/smtp"
)

//go:embed "all:template"
var files embed.FS

const templateDir = "template"

var templateFiles = fstack.New(dev.RelDirFS(templateDir), errsx.Must(fs.Sub(files, templateDir)))

var httpClient = http.Client{Timeout: 10 * time.Second}

type ViewVarsFunc func(r *http.Request) (handler.Vars, error)

type Handler struct {
	*handler.Handler
	signInPath    func() string
	mux           *router.ServeMux
	funcs         template.FuncMap
	viewVarsFuncs map[string]ViewVarsFunc
	Plain         *Renderer
	HTML          *Renderer
	JSON          *Renderer
}

func NewHandler(base *handler.Handler, mux *router.ServeMux, signInPath func() string) *Handler {
	funcs := template.FuncMap{
		"Add":           handler.TmplAdd,
		"Sub":           handler.TmplSub,
		"Mul":           handler.TmplMul,
		"Div":           handler.TmplDiv,
		"Mod":           handler.TmplMod,
		"Ints":          handler.TmplInts,
		"StatusText":    handler.TmplStatusText,
		"Path":          handler.TmplPath(mux),
		"QueryString":   handler.TmplQueryString,
		"FormatTime":    handler.TmplFormatTime,
		"HasPrefix":     handler.TmplHasPrefix,
		"HasSuffix":     handler.TmplHasSuffix,
		"HasPathPrefix": handler.TmplHasPathPrefix(mux),
		"HasString":     handler.TmplHasString,
		"ToStrings":     handler.TmplToStrings,
		"Join":          handler.TmplJoin,
		"ReplaceAll":    handler.TmplReplaceAll,
		"MarshalJSON":   handler.TmplMarshalJSON,
		"UnescapeHTML":  handler.TmplUnescapeHTML,
		"UnescapeJS":    handler.TmplUnescapeJS,
		"Slice":         handler.TmplSlice,
		"Map":           handler.TmplMap,
	}

	h := Handler{
		Handler:       base,
		signInPath:    signInPath,
		viewVarsFuncs: make(map[string]ViewVarsFunc),
		mux:           mux,
		funcs:         funcs,
	}

	h.Plain = NewRenderer(&h, "text/plain")
	h.HTML = NewRenderer(&h, "text/html")
	h.JSON = NewRenderer(&h, "application/json")

	return &h
}

func (h *Handler) SendEmail(ctx context.Context, from, to string, view string, vars handler.Vars) error {
	data := struct {
		URL  handler.URL
		App  handler.AppData
		Vars handler.Vars
	}{
		URL: handler.URL{
			Scheme:   h.Scheme,
			Host:     h.Host,
			Hostname: h.Hostname,
			Port:     h.Port,
		},
		App: handler.AppData{
			Name:        app.Name,
			ShortName:   app.ShortName,
			Description: app.Description,
			ThemeColour: app.ThemeColour,
		},
		Vars: vars,
	}

	var buf bytes.Buffer
	var subject, plain, html string
	email := h.Template(templateFiles, h.funcs, view, "email/"+view+".tmpl")
	for _, view := range []string{"subject", "plain", "html"} {
		tmpl := email.Lookup(view)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute email template: %w", err)
		}

		switch view {
		case "subject":
			subject = buf.String()

		case "plain":
			plain = buf.String()

		case "html":
			html = buf.String()
		}
	}

	msg := smtp.Msg{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Plain:   plain,
		HTML:    html,
	}

	return h.Email.Send(ctx, msg)
}

func (h *Handler) SendSMS(ctx context.Context, to, body string) error {
	config, err := h.Repo.System.FindConfig(ctx)
	if err != nil {
		return fmt.Errorf("find config: %w", err)
	}

	// TODO: Reuse client for as long as Twilio config hasn't changed
	messager := sms.NewTwilioClient(&httpClient, config.TwilioSID, config.TwilioToken)

	return messager.Send(ctx, config.TwilioFromTel, to, body)
}

func (h *Handler) SendTOTPSMS(email, tel string) error {
	ctx := context.Background()

	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	totp, err := user.GenerateTOTP()
	if err != nil {
		return fmt.Errorf("generate TOTP: %w", err)
	}

	if tel == "" {
		tel = user.TOTPTel
	}

	return h.SendSMS(ctx, tel, totp)
}

func (h *Handler) HasPathPrefix(value string, name string, paramArgPairs ...any) bool {
	p := h.Path(name, paramArgPairs...)
	p = strings.TrimSuffix(p, "/")

	return value == p || strings.HasPrefix(value, p+"/")
}

func (h *Handler) Path(name string, paramArgPairs ...any) string {
	return h.mux.Path(name, paramArgPairs...)
}

func (h *Handler) PathQuery(r *http.Request, name string, paramArgPairs ...any) string {
	q := r.URL.Query().Encode()
	if q != "" && !strings.HasPrefix(q, "?") {
		q = "?" + q
	}

	return h.mux.Path(name, paramArgPairs...) + q
}

func (h *Handler) SetViewVars(name string, vars ViewVarsFunc) {
	if _, ok := h.viewVarsFuncs[name]; ok {
		panic(fmt.Sprintf("default view vars already set for %q", name))
	}

	h.viewVarsFuncs[name] = vars
}

func (h *Handler) AddFlashf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.Flash)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.Flash, flash)
}

func (h *Handler) AddFlashImportantf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.FlashImportant)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.FlashImportant, flash)
}

func (h *Handler) AddFlashErrorf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.FlashError)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.FlashError, flash)
}

type PredicateFunc func(p guard.Passport) bool

func (h *Handler) RequireSignIn(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

	if !isSignedIn {
		h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

		http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

		return false
	}

	return true
}

func (h *Handler) RequireSignInIf(check PredicateFunc) router.BeforeHookFunc {
	return func(w http.ResponseWriter, r *http.Request) bool {
		ctx := r.Context()
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
		passport := h.Passport(ctx)

		if !isSignedIn && check(passport) {
			h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

			http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

			return false
		}

		return true
	}
}

func (h *Handler) RequireAuth(check PredicateFunc) router.BeforeHookFunc {
	return func(w http.ResponseWriter, r *http.Request) bool {
		ctx := r.Context()
		passport := h.Passport(ctx)

		if !check(passport) {
			h.HTML.ErrorView(w, r, "require auth", app.ErrUnauthorised, "site/error", nil)

			return false
		}

		return true
	}
}
