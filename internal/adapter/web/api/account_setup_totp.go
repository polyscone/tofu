package api

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"image/jpeg"
	"net/http"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/port/account"
)

func (api *API) accountSetupTOTPPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cmd := account.SetupTOTP{
		Guard:  api.passport(ctx),
		UserID: api.sessions.GetString(ctx, sess.UserID),
	}
	res, err := cmd.Execute(ctx, api.bus)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	keyBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(res.Key)
	issuer := app.Name
	accountName := api.sessions.GetString(ctx, sess.Email)
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
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	qrcode, err = barcode.Scale(qrcode, 200, 200)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, qrcode, nil)
	if writeError(w, r, errors.Tracef(err)) {
		return
	}

	qrcodeBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	writeJSON(w, r, map[string]any{
		"keyBase32":     keyBase32,
		"qrcodeBase64":  qrcodeBase64,
		"recoveryCodes": res.RecoveryCodes,
	})
}
