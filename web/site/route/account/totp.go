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
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/sms"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterTOTPHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Named("account.totp.section", "/account/totp")

	mux.HandleFunc("GET /account/totp/reset", totpResetGet(h), "account.totp.reset")
	mux.HandleFunc("POST /account/totp/reset", totpResetPost(h), "account.totp.reset.post")

	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				user := h.User(ctx)

				if len(user.HashedPassword) == 0 {
					h.AddFlashf(ctx, i18n.M("site.account.totp.flash.password_required"))

					h.Session.SetRedirect(ctx, r.URL.String())

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

		mux.HandleFunc("GET /account/totp/setup/activate", totpSetupActivateGet(h))
		mux.HandleFunc("POST /account/totp/setup/activate", totpSetupActivatePost(h), "account.totp.setup.activate.post")

		mux.HandleFunc("GET /account/totp/setup/success", totpSetupSuccessGet(h), "account.totp.setup.success")

		mux.HandleFunc("GET /account/totp/disable", totpDisableGet(h), "account.totp.disable")
		mux.HandleFunc("POST /account/totp/disable", totpDisablePost(h), "account.totp.disable.post")

		mux.HandleFunc("GET /account/totp/disable/success", totpDisableSuccessGet(h), "account.totp.disable.success")

		mux.HandleFunc("GET /account/totp/recovery-codes", totpRecoveryCodesGet(h), "account.totp.recovery_codes")
		mux.HandleFunc("POST /account/totp/recovery-codes", totpRecoveryCodesPost(h), "account.totp.recovery_codes.post")

		mux.HandleFunc("POST /account/totp/send-sms", totpSendSMSPost(h), "account.totp.sms.send_passcode.post")
	})
}

func totpResetGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/reset/reset", nil)
	}
}

func totpResetPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token    string `form:"token"`
			Password string `form:"password"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindResetTOTPTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find reset TOTP token email", err, "error", nil)

			return
		}

		user, err := h.Repo.Account.FindUserByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by email", err, "error", nil)

			return
		}

		passport, err := h.PassportByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "passport by email", err, "error", nil)

			return
		}

		_, err = h.Svc.Account.ResetTOTP(ctx, passport.Account, user.ID, input.Password)
		if err != nil {
			h.HTML.ErrorViewFunc(w, r, "reset TOTP", err, h.Session.LastView(ctx), func(data *handler.ViewData) error {
				data.ErrorMessage = h.T(ctx, i18n.M("site:account.sign_in.error"))

				return nil
			})

			return
		}

		err = h.Repo.Web.ConsumeResetTOTPToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume reset TOTP token", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.totp.flash.disabled"))

		signInWithPassword(ctx, h, w, r, email, input.Password)
	}
}

func totpSetupGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/setup/methods", nil)
	}
}

func totpSetupPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Method string `form:"method"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		if input.Method != "app" && input.Method != "sms" {
			h.HTML.ErrorView(w, r, "TOTP setup", app.ErrBadRequest, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, err := h.Svc.Account.SetupTOTP(ctx, passport.Account, user.ID)
		if err != nil {
			h.HTML.ErrorView(w, r, "setup TOTP", err, "error", nil)

			return
		}

		switch input.Method {
		case "app":
			http.Redirect(w, r, h.Path("account.totp.setup.app"), http.StatusSeeOther)

		case "sms":
			http.Redirect(w, r, h.Path("account.totp.setup.sms"), http.StatusSeeOther)

		default:
			h.HTML.ErrorView(w, r, "TOTP setup", app.ErrBadRequest, "error", nil)
		}
	}
}

func totpSetupAppGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/totp/setup/app"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()
		user := h.User(ctx)

		keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(user.TOTPKey)
		issuer := app.Name
		accountName := h.Session.Email(ctx)
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

		if h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func totpSetupAppPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, codes, err := h.Svc.Account.VerifyTOTP(ctx, passport.Account, account.VerifyTOTPInput{
			UserID:     user.ID,
			TOTP:       input.TOTP,
			TOTPMethod: "app",
		})
		if err != nil {
			h.HTML.ErrorView(w, r, "verify TOTP", err, h.Session.LastView(ctx), nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}

func totpSetupSMSGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/totp/setup/sms"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
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
			h.HTML.ErrorView(w, r, "TOTP setup SMS", app.ErrNotFound, "error", nil)

			return
		}

		if !user.HasSetupTOTP() {
			http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)

			return
		}

		if h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func totpSetupSMSPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Tel string `form:"tel"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP setup SMS", app.ErrForbidden, "error", nil)

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

			h.HTML.ErrorView(w, r, "send TOTP SMS", err, h.Session.LastView(ctx), nil)

			return
		}

		_, err = h.Svc.Account.ChangeTOTPTel(ctx, passport.Account, user.ID, input.Tel)
		if err != nil {
			h.HTML.ErrorView(w, r, "change TOTP tel", err, h.Session.LastView(ctx), nil)

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
			h.HTML.ErrorView(w, r, "TOTP verify SMS", app.ErrNotFound, "error", nil)

			return
		}

		if !user.HasSetupTOTP() {
			http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)

			return
		}

		if h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/setup/sms_verify", nil)
	}
}

func totpSetupSMSVerifyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if !config.TOTPSMSEnabled {
			h.HTML.ErrorView(w, r, "TOTP verify SMS", app.ErrForbidden, "error", nil)

			return
		}

		_, codes, err := h.Svc.Account.VerifyTOTP(ctx, passport.Account, account.VerifyTOTPInput{
			UserID:     user.ID,
			TOTP:       input.TOTP,
			TOTPMethod: "sms",
		})
		if err != nil {
			h.HTML.ErrorView(w, r, "verify TOTP", err, h.Session.LastView(ctx), nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}

func totpSendSMSPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)

		h.Broker.Dispatch(ctx, event.TOTPSMSRequested{
			Email: user.Email,
			Tel:   user.TOTPTel,
		})

		h.AddFlashf(ctx, i18n.M("site.account.totp.flash.passcode_sms_sent"))

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}

func totpSetupActivateGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, h.Path("account.totp.setup"), http.StatusSeeOther)
	}
}

func totpSetupActivatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, err := h.Svc.Account.ActivateTOTP(ctx, passport.Account, user.ID)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate TOTP", err, "error", nil)

			return
		}

		h.Session.SetTOTPMethod(ctx, user.TOTPMethod)
		h.Session.SetHasActivatedTOTP(ctx, true)

		http.Redirect(w, r, h.Path("account.totp.setup.success"), http.StatusSeeOther)
	}
}

func totpSetupSuccessGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/setup/success", nil)
	}
}

func totpDisableGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/disable/disabled", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/totp/disable/verify", nil)
	}
}

func totpDisablePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Password string `form:"password"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		if config.TOTPRequired {
			h.HTML.ErrorView(w, r, "disable TOTP", app.ErrForbidden, "error", nil)

			return
		}

		_, err := h.Svc.Account.DisableTOTP(ctx, passport.Account, user.ID, input.Password)
		if err != nil {
			if errors.Is(err, account.ErrInvalidPassword) {
				err = fmt.Errorf("%w: %w", app.ErrInvalidInput, errsx.Map{
					"password": errors.New("invalid password"),
				})
			}

			h.HTML.ErrorViewFunc(w, r, "disable TOTP", err, h.Session.LastView(ctx), nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "error", nil)

			return
		}

		h.Session.SetTOTPMethod(ctx, "")
		h.Session.SetHasActivatedTOTP(ctx, false)

		http.Redirect(w, r, h.Path("account.totp.disable.success"), http.StatusSeeOther)
	}
}

func totpDisableSuccessGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/totp/disable/success", nil)
	}
}

func totpRecoveryCodesGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/totp/recovery_codes/regenerate"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
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

		if !h.Session.HasActivatedTOTP(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/totp/recovery_codes/setup_required", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func totpRecoveryCodesPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string `form:"totp"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		user := h.User(ctx)
		passport := h.Passport(ctx)

		_, codes, err := h.Svc.Account.RegenerateRecoveryCodes(ctx, passport.Account, user.ID, input.TOTP)
		if err != nil {
			h.HTML.ErrorView(w, r, "regenerate recovery codes", err, h.Session.LastView(ctx), nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, h.Session.LastView(ctx), handler.Vars{
			"RecoveryCodes": codes,
		})
	}
}
