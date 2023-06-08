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
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/sms"
)

func TOTP(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/totp", func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.Prefix("/setup", func(mux *router.ServeMux) {
			mux.Get("/", totpSetupGet(h), "account.totp.setup")
			mux.Post("/", totpSetupPost(h), "account.totp.setup.post")

			mux.Prefix("/app", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupAppGet(h), "account.totp.setup.app")
				mux.Post("/", totpSetupAppPost(h), "account.totp.setup.app.post")
			})

			mux.Prefix("/sms", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupSMSGet(h), "account.totp.setup.sms")
				mux.Post("/", totpSetupSMSPost(h), "account.totp.setup.sms.post")

				mux.Prefix("/verify", func(mux *router.ServeMux) {
					mux.Get("/", totpSetupSMSVerifyGet(h), "account.totp.setup.sms.verify")
					mux.Post("/", totpSetupSMSVerifyPost(h), "account.totp.setup.sms.verify.post")
				})
			})

			mux.Prefix("/activate", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupActivateGet(h), "account.totp.setup.activate")
				mux.Post("/", totpSetupActivatePost(h), "account.totp.setup.activate.post")
			})

			mux.Get("/success", totpSetupSuccessGet(h), "account.totp.setup.success")
		})

		mux.Prefix("/disable", func(mux *router.ServeMux) {
			mux.Get("/", totpDisableGet(h), "account.totp.disable")
			mux.Post("/", totpDisablePost(h), "account.totp.disable.post")

			mux.Get("/success", totpDisableSuccessGet(h), "account.totp.disable.success")
		})

		mux.Prefix("/recovery-codes", func(mux *router.ServeMux) {
			mux.Get("/", totpRecoveryCodesGet(h), "account.totp.recovery_codes")
			mux.Post("/", totpRecoveryCodesPost(h), "account.totp.recovery_codes.post")
		})

		mux.Post("/send-sms", totpSendSMSPost(h), "account.totp.sms.send_passcode.post")
	})
}

func totpSetupGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/setup/methods", nil)
	}
}

func totpSetupPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Method string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if input.Method != "app" && input.Method != "sms" {
			h.ErrorView(w, r, errors.Tracef(app.ErrBadRequest), "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.SetupTOTP(ctx, passport, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		switch input.Method {
		case "app":
			http.Redirect(w, r, h.Path("account.totp.setup.app"), http.StatusSeeOther)

		case "sms":
			http.Redirect(w, r, h.Path("account.totp.setup.sms"), http.StatusSeeOther)

		default:
			h.ErrorView(w, r, errors.Tracef(app.ErrBadRequest), "error", nil)
		}
	}
}

func totpSetupAppGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/totp/setup/app", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

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
			return nil, errors.Tracef(err)
		}

		qrcode, err = barcode.Scale(qrcode, 200, 200)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, qrcode, nil)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"KeyBase32":    keyBase32,
			"QRCodeBase64": template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())),
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/setup/app", nil)
	}
}

func totpSetupAppPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.VerifyTOTP(ctx, passport, userID, input.TOTP, "app")
		if h.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/app", nil) {
			return
		}

		http.Redirect(w, r, h.Path("account.totp.setup.activate"), http.StatusSeeOther)
	}
}

func totpSetupSMSGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/totp/setup/sms", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"TOTPTelephone": user.TOTPTelephone,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/setup/sms", nil)
	}
}

func totpSetupSMSPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Telephone string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		// We try to send the TOTP SMS first because we don't want to save
		// a telephone number that the SMS provider thinks is invalid
		err = h.SendTOTPSMS(user.Email, input.Telephone)
		if err != nil {
			if errors.Is(err, sms.ErrInvalidNumber) {
				errs := errors.Map{"new telephone": errors.New("invalid phone number")}

				err = errs.Tracef(app.ErrInvalidInput)
			}

			h.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms", nil)

			return
		}

		err = h.Account.ChangeTOTPTelephone(ctx, passport, userID, input.Telephone)
		if h.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms", nil) {
			return
		}

		http.Redirect(w, r, h.Path("account.totp.setup.sms.verify"), http.StatusSeeOther)
	}
}

func totpSetupSMSVerifyGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/setup/sms_verify", nil)
	}
}

func totpSetupSMSVerifyPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.VerifyTOTP(ctx, passport, userID, input.TOTP, "sms")
		if h.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms_verify", nil) {
			return
		}

		http.Redirect(w, r, h.Path("account.totp.setup.activate"), http.StatusSeeOther)
	}
}

func totpSendSMSPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		background.Go(func() {
			if err := h.SendTOTPSMS(user.Email, user.TOTPTelephone); err != nil {
				logger.PrintError(errors.Tracef(err))
			}
		})

		h.AddFlashf(ctx, "A passcode has been sent to your registered phone number.")

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}

func totpSetupActivateGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": user.RecoveryCodes,
		})
	}
}

func totpSetupActivatePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		passport := h.Passport(ctx)

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = h.Account.ActivateTOTP(ctx, passport, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
		h.Sessions.Set(ctx, sess.HasActivatedTOTP, true)

		http.Redirect(w, r, h.Path("account.totp.setup.success"), http.StatusSeeOther)
	}
}

func totpSetupSuccessGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/totp/setup/success", nil)
	}
}

func totpDisableGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/disable/disabled", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/disable/verify", nil)
	}
}

func totpDisablePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Password string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.DisableTOTP(ctx, passport, userID, input.Password)
		if err != nil {
			h.ErrorViewFunc(w, r, errors.Tracef(err), "account/totp/disable/verify", func(data *handler.ViewData) {
				if errors.Is(err, app.ErrBadRequest) {
					data.Errors.Set("password", "invalid password")
				}
			})

			return
		}

		_, err = h.RenewSession(ctx)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.Sessions.Set(ctx, sess.TOTPMethod, "")
		h.Sessions.Set(ctx, sess.HasActivatedTOTP, false)

		http.Redirect(w, r, h.Path("account.totp.disable.success"), http.StatusSeeOther)
	}
}

func totpDisableSuccessGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/totp/disable/success", nil)
	}
}

func totpRecoveryCodesGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/totp/recovery_codes/regenerate", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := h.Sessions.GetInt(ctx, sess.UserID)
		user, err := h.Store.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"RecoveryCodes":     user.RecoveryCodes,
			"ShowRecoveryCodes": false,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !h.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			h.View(w, r, http.StatusOK, "account/totp/recovery_codes/setup_required", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/totp/recovery_codes/regenerate", nil)
	}
}

func totpRecoveryCodesPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.RegenerateRecoveryCodes(ctx, passport, userID, input.TOTP)
		if h.ErrorView(w, r, errors.Tracef(err), "account/totp/recovery_codes/regenerate", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/totp/recovery_codes/regenerate", handler.Vars{
			"ShowRecoveryCodes": true,
		})
	}
}
