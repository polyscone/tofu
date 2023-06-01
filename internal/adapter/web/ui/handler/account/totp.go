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
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/sms"
)

func TOTP(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Prefix("/totp", func(mux *router.ServeMux) {
		guard.RequireSignInPrefix(mux.CurrentPath())

		mux.Prefix("/setup", func(mux *router.ServeMux) {
			mux.Get("/", totpSetupGet(svc), "account.totp.setup")
			mux.Post("/", totpSetupPost(svc), "account.totp.setup.post")

			mux.Prefix("/app", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupAppGet(svc), "account.totp.setup.app")
				mux.Post("/", totpSetupAppPost(svc), "account.totp.setup.app.post")
			})

			mux.Prefix("/sms", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupSMSGet(svc), "account.totp.setup.sms")
				mux.Post("/", totpSetupSMSPost(svc), "account.totp.setup.sms.post")

				mux.Prefix("/verify", func(mux *router.ServeMux) {
					mux.Get("/", totpSetupSMSVerifyGet(svc), "account.totp.setup.sms.verify")
					mux.Post("/", totpSetupSMSVerifyPost(svc), "account.totp.setup.sms.verify.post")
				})
			})

			mux.Prefix("/activate", func(mux *router.ServeMux) {
				mux.Get("/", totpSetupActivateGet(svc), "account.totp.setup.activate")
				mux.Post("/", totpSetupActivatePost(svc), "account.totp.setup.activate.post")
			})

			mux.Get("/success", totpSetupSuccessGet(svc), "account.totp.setup.success")
		})

		mux.Prefix("/disable", func(mux *router.ServeMux) {
			mux.Get("/", totpDisableGet(svc), "account.totp.disable")
			mux.Post("/", totpDisablePost(svc), "account.totp.disable.post")

			mux.Get("/success", totpDisableSuccessGet(svc), "account.totp.disable.success")
		})

		mux.Prefix("/recovery-codes", func(mux *router.ServeMux) {
			mux.Get("/", totpRecoveryCodesGet(svc), "account.totp.recovery_codes")
			mux.Post("/", totpRecoveryCodesPost(svc), "account.totp.recovery_codes.post")
		})

		mux.Post("/send-sms", totpSendSMSPost(svc), "account.totp.sms.send_passcode.post")
	})
}

func totpSetupGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/setup/methods", nil)
	}
}

func totpSetupPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Method string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if input.Method != "app" && input.Method != "sms" {
			svc.ErrorView(w, r, errors.Tracef(app.ErrBadRequest), "error", nil)

			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		err = svc.Account.SetupTOTP(ctx, passport, passport.UserID())
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		switch input.Method {
		case "app":
			http.Redirect(w, r, svc.Path("account.totp.setup.app"), http.StatusSeeOther)

		case "sms":
			http.Redirect(w, r, svc.Path("account.totp.setup.sms"), http.StatusSeeOther)

		default:
			svc.ErrorView(w, r, errors.Tracef(app.ErrBadRequest), "error", nil)
		}
	}
}

func totpSetupAppGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/totp/setup/app", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
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

		if svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/setup/app", nil)
	}
}

func totpSetupAppPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		err = svc.Account.VerifyTOTP(ctx, passport, passport.UserID(), input.TOTP, "app")
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/app", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.totp.setup.activate"), http.StatusSeeOther)
	}
}

func totpSetupSMSGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/totp/setup/sms", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
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

		if svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/setup/sms", nil)
	}
}

func totpSetupSMSPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Telephone string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		// We try to send the TOTP SMS first because we don't want to save
		// a telephone number that the SMS provider thinks is invalid
		err = svc.SendTOTPSMS(user.Email, input.Telephone)
		if err != nil {
			if errors.Is(err, sms.ErrInvalidNumber) {
				errs := errors.Map{"new telephone": errors.New("invalid phone number")}

				err = errs.Tracef(app.ErrInvalidInput)
			}

			svc.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms", nil)

			return
		}

		err = svc.Account.ChangeTOTPTelephone(ctx, passport, passport.UserID(), input.Telephone)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.totp.setup.sms.verify"), http.StatusSeeOther)
	}
}

func totpSetupSMSVerifyGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/setup/sms_verify", nil)
	}
}

func totpSetupSMSVerifyPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		err = svc.Account.VerifyTOTP(ctx, passport, passport.UserID(), input.TOTP, "sms")
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp/setup/sms_verify", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.totp.setup.activate"), http.StatusSeeOther)
	}
}

func totpSendSMSPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		background.Go(func() {
			if err := svc.SendTOTPSMS(user.Email, user.TOTPTelephone); err != nil {
				logger.PrintError(err)
			}
		})

		svc.Flash(ctx, "A passcode has been sent to your registered phone number.")

		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}

func totpSetupActivateGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/setup/enabled", nil)

			return
		}

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/setup/activate", handler.Vars{
			"RecoveryCodes": user.RecoveryCodes,
		})
	}
}

func totpSetupActivatePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		passport := svc.Passport(ctx)

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = svc.Account.ActivateTOTP(ctx, passport, passport.UserID())
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
		svc.Sessions.Set(ctx, sess.HasActivatedTOTP, true)

		http.Redirect(w, r, svc.Path("account.totp.setup.success"), http.StatusSeeOther)
	}
}

func totpSetupSuccessGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/totp/setup/success", nil)
	}
}

func totpDisableGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if !svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/disable/disabled", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/disable/verify", nil)
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
			svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/totp/disable/verify", func(data *handler.ViewData) {
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

		svc.Sessions.Set(ctx, sess.TOTPMethod, "")
		svc.Sessions.Set(ctx, sess.HasActivatedTOTP, false)

		http.Redirect(w, r, svc.Path("account.totp.disable.success"), http.StatusSeeOther)
	}
}

func totpDisableSuccessGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/totp/disable/success", nil)
	}
}

func totpRecoveryCodesGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/totp/recovery_codes/regenerate", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID := svc.Sessions.GetInt(ctx, sess.UserID)
		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
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

		if !svc.Sessions.GetBool(ctx, sess.HasActivatedTOTP) {
			svc.View(w, r, http.StatusOK, "account/totp/recovery_codes/setup_required", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/recovery_codes/regenerate", nil)
	}
}

func totpRecoveryCodesPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		err = svc.Account.RegenerateRecoveryCodes(ctx, passport, passport.UserID(), input.TOTP)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp/recovery_codes/regenerate", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/totp/recovery_codes/regenerate", handler.Vars{
			"ShowRecoveryCodes": true,
		})
	}
}
