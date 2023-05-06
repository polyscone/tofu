package account

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
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

func TOTP(svc *handler.Services, mux *router.ServeMux) {
	mux.Post("/totp", setupTOTPPost(svc))
	mux.Post("/totp/disable", disableTOTPPost(svc))
	mux.Post("/totp/verify", verifyTOTPPost(svc))
	mux.Put("/totp/recovery-codes", regenerateRecoveryCodesPut(svc))
}

func setupTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.SetupTOTP{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
		}
		res, err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
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
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		qrcode, err = barcode.Scale(qrcode, 200, 200)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		var buf bytes.Buffer
		err = jpeg.Encode(&buf, qrcode, nil)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		qrcodeBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

		svc.JSON(w, r, map[string]any{
			"keyBase32":     keyBase32,
			"qrcodeBase64":  qrcodeBase64,
			"recoveryCodes": res.RecoveryCodes,
		})
	}
}

func disableTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.DisableTOTP{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
			TOTP:   input.TOTP,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = passport.Renew()
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		passport.Set(sess.HasVerifiedTOTP, false)

		csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

		svc.JSON(w, r, map[string]any{
			"csrfToken": csrfTokenBase64,
		})
	}
}

func verifyTOTPPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			TOTP string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.VerifyTOTP{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
			TOTP:   input.TOTP,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = passport.Renew()
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		passport.Set(sess.HasVerifiedTOTP, true)

		csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

		svc.JSON(w, r, map[string]any{
			"csrfToken": csrfTokenBase64,
		})
	}
}

func regenerateRecoveryCodesPut(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.RegenerateRecoveryCodes{
			Guard:  passport,
			UserID: passport.GetString(sess.UserID),
		}
		res, err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.JSON(w, r, map[string]any{
			"recoveryCodes": res.RecoveryCodes,
		})
	}
}
