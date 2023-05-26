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
)

func TOTP(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Get("/totp", totpGet(svc), "account.totp")
	mux.Post("/totp", totpPost(svc), "account.totp.post")
	mux.Get("/totp/disable", totpDisableGet(svc), "account.totp.disable")
	mux.Post("/totp/disable", totpDisablePost(svc), "account.totp.disable.post")
	mux.Get("/totp/recovery-codes", totpRecoveryCodesGet(svc), "account.totp.recovery_codes")
	mux.Post("/totp/recovery-codes", totpRecoveryCodesPost(svc), "account.totp.recovery_codes.post")

	guard.ProtectPrefix(svc.Path("account.totp"))

	svc.SetViewVars("account/totp", handler.Vars{
		"RecoveryCodes": nil,
		"KeyBase32":     "",
		"QRCodeBase64":  "",
		"TOTPTelephone": "",
	})

	svc.SetViewVars("account/totp_recovery_codes", handler.Vars{
		"RecoveryCodes": nil,
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
		}
		err := httputil.DecodeForm(&input, r)
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
			err := svc.Account.SetupTOTP(ctx, passport, passport.UserID())
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			http.Redirect(w, r, svc.Path("account.totp")+"?method="+method, http.StatusSeeOther)

		case "send-sms":
			err := svc.Account.ChangeTOTPTelephone(ctx, passport, passport.UserID(), input.Telephone)
			if err != nil {
				totpDisplay(svc, w, r, err)

				return
			}

			err = svc.SendTOTPSMS(svc.Sessions.GetString(ctx, sess.Email))

			totpDisplay(svc, w, r, err)

		case "verify":
			err := svc.Account.VerifyTOTP(ctx, passport, passport.UserID(), input.TOTP, method)
			if err != nil {
				totpDisplay(svc, w, r, err)

				return
			}

			svc.Sessions.Set(ctx, sess.HasVerifiedTOTP, true)
			svc.Sessions.Set(ctx, sess.TOTPMethod, method)

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

	userID := svc.Sessions.GetInt(ctx, sess.UserID)
	user, err := svc.Repo.Account.FindUserByID(ctx, userID)
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

	userID := svc.Sessions.GetInt(ctx, sess.UserID)
	user, err := svc.Repo.Account.FindUserByID(ctx, userID)
	if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
		return
	}

	vars := handler.Vars{
		"RecoveryCodes": user.RecoveryCodes,
		"TOTPTelephone": user.TOTPTelephone,
	}

	if errors.Is(_err, sms.ErrInvalidNumber) {
		errs := errors.Map{"new telephone": errors.New("invalid phone number")}

		_err = errs.Tracef(app.ErrInvalidInput)
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
			Password string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		err = svc.Account.DisableTOTP(ctx, passport, passport.UserID(), input.Password)
		if err != nil {
			svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/totp_disable", func(data *handler.ViewData) {
				if errors.Is(err, app.ErrBadRequest) {
					data.Errors.Set("password", "invalid password")
				}
			})

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

func totpRecoveryCodesGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/totp_recovery_codes", handler.Vars{
			"RecoveryCodes": user.RecoveryCodes,
		})
	}
}

func totpRecoveryCodesPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		passport := svc.Passport(ctx)

		err := svc.Account.RegenerateRecoveryCodes(ctx, passport, passport.UserID())
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.totp.recovery_codes")+"?status=success", http.StatusSeeOther)
	}
}
