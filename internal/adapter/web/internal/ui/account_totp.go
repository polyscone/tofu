package ui

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
	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (ui *UI) accountTOTPGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account_totp", nil)
}

func (ui *UI) accountTOTPSetupAppPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cmd := account.SetupTOTP{
		Guard:  ui.passport(ctx),
		UserID: ui.sessions.GetString(ctx, sesskey.UserID),
	}
	res, err := cmd.Execute(ctx, ui.bus)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(res.Key)
	issuer := app.Name
	accountName := ui.sessions.GetString(ctx, sesskey.Email)
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
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	qrcode, err = barcode.Scale(qrcode, 200, 200)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, qrcode, nil)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	ui.render(w, r, http.StatusOK, "account_totp", func(data *renderData) {
		data.TOTP.KeyBase32 = keyBase32
		data.TOTP.QRCodeBase64 = template.URL("data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()))
	})
}

func (ui *UI) accountTOTPVerifyPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TOTP string
	}
	if ui.renderError(w, r, errors.Tracef(decodeForm(r, &input))) {
		return
	}

	ctx := r.Context()

	cmd := account.VerifyTOTP{
		Guard:  ui.passport(ctx),
		UserID: ui.sessions.GetString(ctx, sesskey.UserID),
		TOTP:   input.TOTP,
	}
	err := cmd.Execute(ctx, ui.bus)
	if err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Redirect(w, r, ui.route("account.totp")+"?status=failed", http.StatusSeeOther)

		return
	}

	http.Redirect(w, r, ui.route("account.totp")+"?status=success", http.StatusSeeOther)
}
