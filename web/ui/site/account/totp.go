package account

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"image/jpeg"
	"net/http"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/sms"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/sess"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterTOTPHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Named("account.totp.section", "/account/totp")

	mux.HandleFunc("GET /account/totp/reset", h.HTML.HandlerFunc("site/account/totp/reset/reset"), "account.totp.reset")
	mux.HandleFunc("POST /account/totp/reset", totpResetPost(h), "account.totp.reset.post")

	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				user := h.User(ctx)

				if len(user.HashedPassword) == 0 {
					h.AddFlashf(ctx, "You need to choose a password before you can setup two-factor authentication.")

					h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

					http.Redirect(w, r, h.Path("account.choose_password"), http.StatusSeeOther)

					return
				}

				next(w, r)
			}
		})

		mux.HandleFunc("GET /account/totp/setup", totpSetupGet(h), "account.totp.setup")
		mux.HandleFunc("POST /account/totp/setup", totpSetupPost(h), "account.totp.setup.post")

		mux.HandleFunc("GET /account/totp/setup/app", totpSetupAppGet(h), "account.totp.setup.app")
		mux.HandleFunc("POST /account/totp/setup/app", totpSetupAppPost(h), "account.totp.setup.app.post")

		mux.HandleFunc("GET /account/totp/setup/sms", totpSetupSMSGet(h), "account.totp.setup.sms")
		mux.HandleFunc("POST /account/totp/setup/sms", totpSetupSMSPost(h), "account.totp.setup.sms.post")

		mux.HandleFunc("GET /account/totp/setup/sms/verify", totpSetupSMSVerifyGet(h), "account.totp.setup.sms.verify")
		mux.HandleFunc("POST /account/totp/setup/sms/verify", totpSetupSMSVerifyPost(h), "account.totp.setup.sms.verify.post")

		mux.HandleFunc("POST /account/totp/setup/activate", totpSetupActivatePost(h), "account.totp.setup.activate.post")

		mux.HandleFunc("GET /account/totp/setup/success", h.HTML.HandlerFunc("site/account/totp/setup/success"), "account.totp.setup.success")

		mux.HandleFunc("GET /account/totp/disable", totpDisableGet(h), "account.totp.disable")
		mux.HandleFunc("POST /account/totp/disable", totpDisablePost(h), "account.totp.disable.post")

		mux.HandleFunc("GET /account/totp/disable/success", h.HTML.HandlerFunc("site/account/totp/disable/success"), "account.totp.disable.success")

		mux.HandleFunc("GET /account/totp/recovery-codes", totpRecoveryCodesGet(h), "account.totp.recovery_codes")
		mux.HandleFunc("POST /account/totp/recovery-codes", totpRecoveryCodesPost(h), "account.totp.recovery_codes.post")

		mux.HandleFunc("POST /account/totp/send-sms", totpSendSMSPost(h), "account.totp.sms.send_passcode.post")
	})
}

func totpSetupGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/methods", nil)
	}
}

func totpSetupPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Method string `form:"method"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		if input.Method != "app" && input.Method != "sms" {
			h.HTML.ErrorView(w, r, "TOTP setup", app.ErrBadRequest, "site/error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, err := h.Svc.Account.SetupTOTP(ctx, passport.Account, user.ID)
		if err != nil {
			h.HTML.ErrorView(w, r, "setup TOTP", err, "site/error", nil)

			return
		}

		switch input.Method {
		case "app":
			http.Redirect(w, r, h.Path("account.totp.setup.app"), http.StatusSeeOther)

		case "sms":
			http.Redirect(w, r, h.Path("account.totp.setup.sms"), http.StatusSeeOther)

		default:
			h.HTML.ErrorView(w, r, "TOTP setup", app.ErrBadRequest, "site/error", nil)
		}
	}
}

func totpSetupAppGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("site/account/totp/setup/app", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()
		user := h.User(ctx)

		keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(user.TOTPKey)
		issuer := app.Name
		accountName := h.Sessions.GetString(ctx, sess.Email)
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
		if err != nil {
			return nil, err
		}

		qrcode, err = barcode.Scale(qrcode, 200, 200)
		if err != nil {
			return nil, fmt.Errorf("scale QR code: %w", err)
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, qrcode, nil)
		if err != nil {
			return nil, fmt.Errorf("encode QR code as JPEG: %w", err)
		}

		vars := handler.Vars{
			"KeyBase32":    keyBase32,
			"QRCodeBase64": template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())),
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)

		if !user.HasSetupTOTP() {
			http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/app", nil)
	}
}

func totpSetupAppPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, codes, err := h.Svc.Account.VerifyTOTP(ctx, passport.Account, user.ID, input.TOTP, "app")
		if err != nil {
			h.HTML.ErrorView(w, r, "verify TOTP", err, "site/account/totp/setup/app", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}

func totpSetupSMSGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("site/account/totp/setup/sms", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()
		user := h.User(ctx)

		vars := handler.Vars{
			"TOTPTel": user.TOTPTel,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP setup SMS", app.ErrNotFound, "site/error", nil)

			return
		}

		if !user.HasSetupTOTP() {
			http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/sms", nil)
	}
}

func totpSetupSMSPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Tel string `form:"tel"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP setup SMS", app.ErrForbidden, "site/error", nil)

			return
		}

		// We try to send the TOTP SMS first because we don't want to save
		// a phone number that the SMS provider thinks is invalid
		err := h.SendTOTPSMS(user.Email, input.Tel)
		if err != nil {
			if errors.Is(err, sms.ErrInvalidNumber) {
				err = fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
					"new phone": errors.New("invalid phone number"),
				})
			}

			h.HTML.ErrorView(w, r, "send TOTP SMS", err, "site/account/totp/setup/sms", nil)

			return
		}

		_, err = h.Svc.Account.ChangeTOTPTel(ctx, passport.Account, user.ID, input.Tel)
		if err != nil {
			h.HTML.ErrorView(w, r, "change TOTP tel", err, "site/account/totp/setup/sms", nil)

			return
		}

		http.Redirect(w, r, h.Path("account.totp.setup.sms.verify"), http.StatusSeeOther)
	}
}

func totpSetupSMSVerifyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP verify SMS", app.ErrNotFound, "site/error", nil)

			return
		}

		if !user.HasSetupTOTP() {
			http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)

			return
		}

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/sms_verify", nil)
	}
}

func totpSetupSMSVerifyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP verify SMS", app.ErrForbidden, "site/error", nil)

			return
		}

		_, codes, err := h.Svc.Account.VerifyTOTP(ctx, passport.Account, user.ID, input.TOTP, "sms")
		if err != nil {
			h.HTML.ErrorView(w, r, "verify TOTP", err, "site/account/totp/setup/sms_verify", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}

func totpSendSMSPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)

		h.Broker.Dispatch(event.TOTPSMSRequested{
			Email: user.Email,
			Tel:   user.TOTPTel,
		})

		h.AddFlashf(ctx, "A passcode has been sent to your registered phone number.")

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}

func totpSetupActivatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, err := h.Svc.Account.ActivateTOTP(ctx, passport.Account, user.ID)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate TOTP", err, "site/error", nil)

			return
		}

		h.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
		h.Sessions.Set(ctx, sess.HasActivatedTOTP, true)

		http.Redirect(w, r, h.Path("account.totp.setup.success"), http.StatusSeeOther)
	}
}

func totpDisableGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/disable/disabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/disable/verify", nil)
	}
}

func totpDisablePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Password string `form:"password"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if config.TOTPRequired {
			h.HTML.ErrorView(w, r, "disable TOTP", app.ErrForbidden, "site/error", nil)

			return
		}

		_, err := h.Svc.Account.DisableTOTP(ctx, passport.Account, user.ID, input.Password)
		if err != nil {
			if errors.Is(err, account.ErrInvalidPassword) {
				err = fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
					"password": errors.New("invalid password"),
				})
			}

			h.HTML.ErrorViewFunc(w, r, "disable TOTP", err, "site/account/totp/disable/verify", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "site/error", nil)

			return
		}

		h.Sessions.Set(ctx, sess.TOTPMethod, "")
		h.Sessions.Set(ctx, sess.HasActivatedTOTP, false)

		http.Redirect(w, r, h.Path("account.totp.disable.success"), http.StatusSeeOther)
	}
}

func totpResetPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token    string `form:"token"`
			Password string `form:"password"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindResetTOTPTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find reset TOTP token email", err, "site/error", nil)

			return
		}

		user, err := h.Repo.Account.FindUserByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by email", err, "site/error", nil)

			return
		}

		passport, err := h.PassportByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "passport by email", err, "site/error", nil)

			return
		}

		_, err = h.Svc.Account.ResetTOTP(ctx, passport.Account, user.ID, input.Password)
		if err != nil {
			h.HTML.ErrorViewFunc(w, r, "reset TOTP", err, "site/account/totp/reset/reset", func(data *handler.ViewData) error {
				data.ErrorMessage = "Either your credentials are incorrect, or you're not authorised to access this application."

				return nil
			})

			return
		}

		err = h.Repo.Web.ConsumeResetTOTPToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume reset TOTP token", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Two-factor authentication has been disabled for your account.")

		signInWithPassword(ctx, h, w, r, email, input.Password)
	}
}

func totpRecoveryCodesGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("site/account/totp/recovery_codes/regenerate", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()
		user := h.User(ctx)

		vars := handler.Vars{
			"HashedRecoveryCodes": user.HashedRecoveryCodes,
			"RecoveryCodes":       nil,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.HTML.View(w, r, http.StatusOK, "site/account/totp/recovery_codes/setup_required", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/recovery_codes/regenerate", nil)
	}
}

func totpRecoveryCodesPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, codes, err := h.Svc.Account.RegenerateRecoveryCodes(ctx, passport.Account, user.ID, input.TOTP)
		if err != nil {
			h.HTML.ErrorView(w, r, "regenerate recovery codes", err, "site/account/totp/recovery_codes/regenerate", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/totp/recovery_codes/regenerate", handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}
