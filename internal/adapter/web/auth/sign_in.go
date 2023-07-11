package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

var client = http.Client{Timeout: 10 * time.Second}

func SignInWithPassword(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email, password string) error {
	logger := h.Logger(ctx)

	attempts := h.Sessions.GetInt(ctx, sess.SignInAttempts)
	lastAttemptAt := h.Sessions.GetTime(ctx, sess.LastSignInAttemptAt)
	if time.Since(lastAttemptAt) > app.SignInThrottleTTL {
		attempts = 0
		lastAttemptAt = time.Time{}
	}

	if err := h.Svc.Account.CheckSignInThrottle(attempts, lastAttemptAt); err != nil {
		return fmt.Errorf("check session sign in throttle: %w", err)
	}

	err := h.Svc.Account.SignInWithPassword(ctx, email, password)
	if err != nil {
		attempts++
		lastAttemptAt = time.Now().UTC()

		h.Sessions.Set(ctx, sess.SignInAttempts, attempts)
		h.Sessions.Set(ctx, sess.LastSignInAttemptAt, lastAttemptAt)

		return err
	}

	h.Sessions.Delete(ctx, sess.SignInAttempts)
	h.Sessions.Delete(ctx, sess.LastSignInAttemptAt)

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	if err := SignInSetSession(ctx, h, w, r, email); err != nil {
		return fmt.Errorf("sign in set session: %w", err)
	}

	knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(password))
	if err != nil {
		logger.Error("known password breach count", "error", err)
	}
	if knownBreachCount > 0 {
		h.Sessions.Set(ctx, sess.KnownPasswordBreachCount, knownBreachCount)
	}

	return nil
}

func SignInWithTOTP(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, totp string) error {
	user := h.User(ctx)

	err := h.Svc.Account.SignInWithTOTP(ctx, user.ID, totp)
	if err != nil {
		return err
	}

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	h.Sessions.Set(ctx, sess.IsSignedIn, true)
	h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

	return nil
}

func SignInWithRecoveryCode(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, recoveryCode string) error {
	user := h.User(ctx)

	err := h.Svc.Account.SignInWithRecoveryCode(ctx, user.ID, recoveryCode)
	if err != nil {
		return err
	}

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	h.Sessions.Set(ctx, sess.IsSignedIn, true)
	h.Sessions.Delete(ctx, sess.IsAwaitingTOTP)

	return nil
}

func SignInWithGoogle(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, jwt string) error {
	config := h.Config(ctx)

	if !config.GoogleSignInEnabled {
		return errors.New("Google sign in is disabled")
	}
	if config.GoogleSignInClientID == "" {
		return errors.New("Google sign in client id has not be set")
	}

	// TODO: Check cache-control
	res, err := client.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return fmt.Errorf("fetch Google OAuth2 certs: %w", err)
	}
	defer res.Body.Close()

	certs := make(map[string]string)
	if err := httputil.DecodeJSON(&certs, res.Body); err != nil {
		return fmt.Errorf("decode Google OAuth2 certs JSON: %w", err)
	}

	parts := strings.Split(jwt, ".")
	if want, got := 3, len(parts); want != got {
		return fmt.Errorf("want %v parts in JWT; got %v", want, got)
	}

	var header struct {
		Alg string
		Kid string // Key ID to use from Google's public keys
		Typ string
	}
	if b, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return fmt.Errorf("decode JWT header: %w", err)
	} else if json.Unmarshal(b, &header); err != nil {
		return fmt.Errorf("unmarshal JWT header: %w", err)
	}

	if header.Typ != "JWT" {
		return fmt.Errorf("check JWT header: want JWT type; got %q", header.Typ)
	}
	if header.Alg != "RS256" {
		return fmt.Errorf("check JWT header: want RS256 algorithm; got %q", header.Alg)
	}

	block, _ := pem.Decode([]byte(certs[header.Kid]))
	if block == nil {
		return fmt.Errorf("unable to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse Google OAuth2 cert: %w", err)
	}

	rsaPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("could not assert cert.PublicKey as %T", rsaPublicKey)
	}

	payload := sha256.New()
	if _, err := payload.Write([]byte(parts[0] + "." + parts[1])); err != nil {
		return fmt.Errorf("new JWT payload hash: %w", err)
	}
	hashed := payload.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decode JWT signature: %w", err)
	}

	if err := rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed, signature); err != nil {
		return fmt.Errorf("check JWT signature: %w", err)
	}

	var claims struct {
		Aud   string // Client ID
		Iss   string // accounts.google.com or https://accounts.google.com
		Exp   int64
		Nbf   int64
		Email string
	}
	if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return fmt.Errorf("decode JWT claims: %w", err)
	} else if json.Unmarshal(b, &claims); err != nil {
		return fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	if claims.Aud != config.GoogleSignInClientID {
		return fmt.Errorf("invalid client id in JWT claims")
	}

	if claims.Iss != "accounts.google.com" && claims.Iss != "https://accounts.google.com" {
		return fmt.Errorf("invalid issuer %q in JWT claims", claims.Iss)
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && claims.Exp <= now {
		return fmt.Errorf("JWT is expired")
	}
	if claims.Nbf > 0 && claims.Nbf > now {
		return fmt.Errorf("JWT used too soon")
	}

	behaviour := account.GoogleSignInOnly
	if config.SignUpEnabled {
		behaviour = account.GoogleAllowSignUp
	}

	if err := h.Svc.Account.SignInWithGoogle(ctx, claims.Email, behaviour); err != nil {
		return fmt.Errorf("sign in wih Google: %w", err)
	}

	if _, err := h.RenewSession(ctx); err != nil {
		return fmt.Errorf("renew session: %w", err)
	}

	if err := SignInSetSession(ctx, h, w, r, claims.Email); err != nil {
		return fmt.Errorf("sign in set session: %w", err)
	}

	return nil
}

func SignInSetSession(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email string) error {
	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	h.Sessions.Set(ctx, sess.UserID, user.ID)
	h.Sessions.Set(ctx, sess.Email, email)
	h.Sessions.Set(ctx, sess.TOTPMethod, user.TOTPMethod)
	h.Sessions.Set(ctx, sess.HasActivatedTOTP, user.HasActivatedTOTP())
	h.Sessions.Set(ctx, sess.IsAwaitingTOTP, user.HasActivatedTOTP())
	h.Sessions.Set(ctx, sess.IsSignedIn, !user.HasActivatedTOTP())

	return nil
}
