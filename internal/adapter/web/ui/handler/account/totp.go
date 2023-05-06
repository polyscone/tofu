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
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func TOTP(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Get("/totp", totpGet(svc), "account.totp")
	mux.Post("/totp/app", totpSetupWithAppPost(svc), "account.totp.app.post")
	mux.Post("/totp/verify", totpVerifyPost(svc), "account.totp.verify.post")
	mux.Get("/totp/disable", totpDisableGet(svc), "account.totp.disable")
	mux.Post("/totp/disable", totpDisablePost(svc), "account.totp.disable.post")

	mux.Redirect(http.MethodGet, svc.Path("account.totp.app.post"), svc.Path("account.totp"), http.StatusSeeOther)

	svc.SetViewVars("account/totp", handler.Vars{
		"RecoveryCodes": nil,
		"KeyBase32":     "",
		"QRCodeBase64":  "",
	})

	guard.Protect(svc.Path("account.totp"))
}

func totpGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/totp", nil)
	}
}

func totpSetupWithAppPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.SetupTOTP{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
		}
		res, err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(res.Key)
		issuer := app.Name
		accountName := passport.GetString(sess.Email)
		qrcode, err := qr.Encode(
			"otpauth://totp/"+
				issuer+":"+accountName+
				"?secret="+keyBase32+
				"&issuer="+issuer+
				"&algorithm="+res.Algorithm+
				"&digits="+strconv.Itoa(res.Digits)+
				"&period="+strconv.Itoa(res.Period),
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

		svc.View(w, r, http.StatusOK, "account/totp", handler.Vars{
			"RecoveryCodes": res.RecoveryCodes,
			"KeyBase32":     keyBase32,
			"QRCodeBase64":  template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())),
		})
	}
}

func totpVerifyPost(svc *handler.Services) http.HandlerFunc {
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

		cmd := account.VerifyTOTP{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
			TOTP:   input.TOTP,
		}
		err = cmd.Execute(ctx, svc.Bus)
		if err != nil {
			httputil.LogError(r, errors.Tracef(err))

			http.Redirect(w, r, svc.Path("account.totp")+"?status=failed", http.StatusSeeOther)

			return
		}

		passport.Set(sess.HasVerifiedTOTP, true)

		http.Redirect(w, r, svc.Path("account.totp")+"?status=success", http.StatusSeeOther)
	}
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
			UserID: passport.GetString(sess.UserID),
			TOTP:   input.TOTP,
		}
		err = cmd.Execute(ctx, svc.Bus)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/totp_disable", nil) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = passport.Renew()
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport.Set(sess.HasVerifiedTOTP, false)

		http.Redirect(w, r, svc.Path("account.totp.disable")+"?status=disabled", http.StatusSeeOther)
	}
}
