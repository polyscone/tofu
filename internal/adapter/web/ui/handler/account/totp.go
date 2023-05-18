package account

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"html/template"
	"image/jpeg"
	"net/http"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
)

func TOTP(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Get("/totp", totpGet(svc), "account.totp")
	mux.Post("/totp", totpPost(svc), "account.totp.post")
	mux.Get("/totp/disable", totpDisableGet(svc), "account.totp.disable")
	mux.Post("/totp/disable", totpDisablePost(svc), "account.totp.disable.post")
	mux.Post("/totp/disable/send-sms", totpDisableSendSMSPost(svc), "account.totp.disable.send_sms.post")

	guard.Protect(svc.Path("account.totp"))
	guard.Protect(svc.Path("account.totp.post"))
	guard.Protect(svc.Path("account.totp.disable"))
	guard.Protect(svc.Path("account.totp.disable.post"))

	svc.SetViewVars("account/totp", handler.Vars{
		"RecoveryCodes": nil,
		"KeyBase32":     "",
		"QRCodeBase64":  "",
		"TOTPTelephone": "",
	})
}

func totpGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("method") {
		case "app":
			totpDisplayApp(svc, w, r, nil)

		case "sms":
			totpDisplaySMS(svc, w, r, nil)

		default:
			svc.View(w, r, http.StatusOK, "account/totp", nil)
		}
	}
}

func totpPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Action    string
			Telephone string
			TOTP      string
			UseSMS    bool
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		actions := map[string]struct{}{
			"setup":    {},
			"send-sms": {},
			"verify":   {},
		}
		if _, ok := actions[input.Action]; !ok {
			svc.ErrorView(w, r, errors.Tracef("invalid action %q", input.Action), "error", nil)

			return
		}

		method := r.URL.Query().Get("method")
		methods := map[string]struct{}{
			"app": {},
			"sms": {},
		}
		if _, ok := methods[method]; !ok {
			svc.ErrorView(w, r, errors.Tracef("invalid method %q", method), "error", nil)

			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		switch input.Action {
		case "setup":
			cmd := account.SetupTOTP{
				Guard:  passport,
				UserID: passport.UserID(),
			}
			_, err := cmd.Execute(ctx, svc.Bus)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			http.Redirect(w, r, svc.Path("account.totp")+"?method="+method, http.StatusSeeOther)

		case "send-sms":
			changeTOTPTelephoneCmd := account.ChangeTOTPTelephone{
				Guard:        passport,
				UserID:       passport.UserID(),
				NewTelephone: input.Telephone,
			}
			err := changeTOTPTelephoneCmd.Execute(ctx, svc.Bus)
			if err != nil {
				totpDisplay(svc, w, r, err)

				return
			}

			err = svc.SendTOTPSMS(svc.Sessions.GetString(ctx, sess.Email))

			totpDisplay(svc, w, r, err)

		case "verify":
			cmd := account.VerifyTOTP{
				Guard:  passport,
				UserID: passport.UserID(),
				TOTP:   input.TOTP,
				UseSMS: input.UseSMS,
			}
			err := cmd.Execute(ctx, svc.Bus)
			if err != nil {
				totpDisplay(svc, w, r, err)

				return
			}

			svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, true)
			svc.Sessions.Set(ctx, sess.TOTPUseSMS, input.UseSMS)

			http.Redirect(w, r, svc.Path("account.totp")+"?status=success", http.StatusSeeOther)
		}
	}
}

func totpDisplay(svc *handler.Services, w http.ResponseWriter, r *http.Request, err error) {
	switch r.URL.Query().Get("method") {
	case "app":
		totpDisplayApp(svc, w, r, err)

	case "sms":
		totpDisplaySMS(svc, w, r, err)

	default:
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/totp", nil)
	}
}

func totpDisplayApp(svc *handler.Services, w http.ResponseWriter, r *http.Request, _err error) {
	ctx := r.Context()

	userID := svc.Sessions.GetString(ctx, sess.UserID)
	cmd := account.FindUserByID{
		UserID: userID,
	}
	user, err := cmd.Execute(ctx, svc.Bus)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(user.TOTPKey)
	issuer := app.Name
	accountName := svc.Sessions.GetString(ctx, sess.Email)
	qrcode, err := qr.Encode(
		"otpauth://totp/"+
			issuer+":"+accountName+
			"?secret="+keyBase32+
			"&issuer="+issuer+
			"&algorithm="+user.TOTPAlgorithm+
			"&digits="+strconv.Itoa(user.TOTPDigits)+
			"&period="+strconv.Itoa(int(user.TOTPPeriod.Seconds())),
		qr.M,
		qr.Auto,
	)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	qrcode, err = barcode.Scale(qrcode, 200, 200)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, qrcode, nil)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	vars := handler.Vars{
		"RecoveryCodes": user.RecoveryCodes,
		"KeyBase32":     keyBase32,
		"QRCodeBase64":  template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())),
	}

	if svc.ErrorView(w, r, errors.Tracef(_err), "account/totp", vars) {
		return
	}

	svc.View(w, r, http.StatusOK, "account/totp", vars)
}

func totpDisplaySMS(svc *handler.Services, w http.ResponseWriter, r *http.Request, _err error) {
	ctx := r.Context()

	userID := svc.Sessions.GetString(ctx, sess.UserID)
	cmd := account.FindUserByID{
		UserID: userID,
	}
	user, err := cmd.Execute(ctx, svc.Bus)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	vars := handler.Vars{
		"RecoveryCodes": user.RecoveryCodes,
		"TOTPTelephone": user.TOTPTelephone,
	}

	if errors.Is(_err, sms.ErrInvalidNumber) {
		errs := errors.Map{"new telephone": errors.New("invalid phone number")}

		_err = errs.Tracef(port.ErrInvalidInput)
	}

	if svc.ErrorView(w, r, errors.Tracef(_err), "account/totp", vars) {
		return
	}

	svc.View(w, r, http.StatusOK, "account/totp", vars)
}

func totpDisableGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/totp_disable", nil)
	}
}

func totpDisablePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.DisableTOTP{
			Guard:  passport,
			UserID: passport.UserID(),
			TOTP:   input.TOTP,
		}
		err = cmd.Execute(ctx, svc.Bus)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp_disable", nil) {
			return
		}

		_, err = svc.RenewSession(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, false)

		http.Redirect(w, r, svc.Path("account.totp.disable")+"?status=disabled", http.StatusSeeOther)
	}
}

func totpDisableSendSMSPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		svc.Broker.Dispatch(handler.TOTPSMSRequested{
			Email: svc.Sessions.GetString(ctx, sess.Email),
		})

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}
