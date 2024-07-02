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

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/password/pwned"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/sess"
)

var client = http.Client{Timeout: 10 * time.Second}

func SignInWithPassword(ctx context.Context, h *handler.Handler, email, password string) error {
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

	if err := SignInSetSession(ctx, h, email); err != nil {
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

func SignInWithMagicLink(ctx context.Context, h *handler.Handler, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("%w: empty token", app.ErrInvalidInput)
	}

	config := h.Config(ctx)

	email, err := h.Repo.Web.FindSignInMagicLinkTokenEmail(ctx, token)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			err = fmt.Errorf("%w: %w", app.ErrInvalidInput, err)
		}

		return false, fmt.Errorf("find sign in magic link token email: %w", err)
	}

	behaviour := account.MagicLinkSignInOnly
	if config.SignUpEnabled {
		if config.SignUpAutoActivateEnabled {
			behaviour = account.MagicLinkAllowSignUpActivate
		} else {
			behaviour = account.MagicLinkAllowSignUp
		}
	}

	signedIn, err := h.Svc.Account.SignInWithMagicLink(ctx, email, behaviour)
	if err != nil {
		return false, fmt.Errorf("sign in wih magic link: %w", err)
	}

	err = h.Repo.Web.ConsumeSignInMagicLinkToken(ctx, token)
	if err != nil {
		return signedIn, fmt.Errorf("consume sign in magic link token: %w", err)
	}

	if signedIn {
		if _, err := h.RenewSession(ctx); err != nil {
			return signedIn, fmt.Errorf("renew session: %w", err)
		}

		if err := SignInSetSession(ctx, h, email); err != nil {
			return signedIn, fmt.Errorf("sign in set session: %w", err)
		}
	}

	return signedIn, nil
}

func SignInWithTOTP(ctx context.Context, h *handler.Handler, totp string) error {
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

func SignInWithRecoveryCode(ctx context.Context, h *handler.Handler, recoveryCode string) error {
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

func SignInWithGoogle(ctx context.Context, h *handler.Handler, jwt string) (bool, error) {
	config := h.Config(ctx)

	if !config.GoogleSignInEnabled {
		return false, errors.New("Google sign in is disabled")
	}
	if config.GoogleSignInClientID == "" {
		return false, errors.New("Google sign in client id is empty")
	}

	// TODO: Check cache-control
	res, err := client.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return false, fmt.Errorf("fetch Google OAuth2 certs: %w", err)
	}
	defer res.Body.Close()

	certs := make(map[string]string)
	if err := httpx.DecodeJSON(&certs, res.Body); err != nil {
		return false, fmt.Errorf("decode Google OAuth2 certs JSON: %w", err)
	}

	parts := strings.Split(jwt, ".")
	if want, got := 3, len(parts); want != got {
		return false, fmt.Errorf("want %v parts in JWT; got %v", want, got)
	}

	var header struct {
		Alg string
		Kid string // Key ID to use from Google's public keys
		Typ string
	}
	if b, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return false, fmt.Errorf("decode JWT header: %w", err)
	} else if err := json.Unmarshal(b, &header); err != nil {
		return false, fmt.Errorf("unmarshal JWT header: %w", err)
	}

	if header.Typ != "JWT" {
		return false, fmt.Errorf("check JWT header: want JWT type; got %q", header.Typ)
	}
	if header.Alg != "RS256" {
		return false, fmt.Errorf("check JWT header: want RS256 algorithm; got %q", header.Alg)
	}

	block, _ := pem.Decode([]byte(certs[header.Kid]))
	if block == nil {
		return false, fmt.Errorf("unable to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("parse Google OAuth2 cert: %w", err)
	}

	rsaPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("could not assert cert.PublicKey as %T", rsaPublicKey)
	}

	payload := sha256.New()
	if _, err := payload.Write([]byte(parts[0] + "." + parts[1])); err != nil {
		return false, fmt.Errorf("new JWT payload hash: %w", err)
	}
	hashed := payload.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("decode JWT signature: %w", err)
	}

	if err := rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed, signature); err != nil {
		return false, fmt.Errorf("check JWT signature: %w", err)
	}

	var claims struct {
		Aud   string // Client ID
		Iss   string // accounts.google.com or https://accounts.google.com
		Exp   int64
		Nbf   int64
		Email string
	}
	if b, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return false, fmt.Errorf("decode JWT claims: %w", err)
	} else if err := json.Unmarshal(b, &claims); err != nil {
		return false, fmt.Errorf("unmarshal JWT claims: %w", err)
	}

	if claims.Aud != config.GoogleSignInClientID {
		return false, fmt.Errorf("invalid client id in JWT claims")
	}

	if claims.Iss != "accounts.google.com" && claims.Iss != "https://accounts.google.com" {
		return false, fmt.Errorf("invalid issuer %q in JWT claims", claims.Iss)
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && claims.Exp <= now {
		return false, fmt.Errorf("JWT is expired")
	}
	if claims.Nbf > 0 && claims.Nbf > now {
		return false, fmt.Errorf("JWT used too soon")
	}

	behaviour := account.GoogleSignInOnly
	if config.SignUpEnabled {
		if config.SignUpAutoActivateEnabled {
			behaviour = account.GoogleAllowSignUpActivate
		} else {
			behaviour = account.GoogleAllowSignUp
		}
	}

	signedIn, err := h.Svc.Account.SignInWithGoogle(ctx, claims.Email, behaviour)
	if err != nil {
		return false, fmt.Errorf("sign in wih Google: %w", err)
	}

	if signedIn {
		if _, err := h.RenewSession(ctx); err != nil {
			return signedIn, fmt.Errorf("renew session: %w", err)
		}

		if err := SignInSetSession(ctx, h, claims.Email); err != nil {
			return signedIn, fmt.Errorf("sign in set session: %w", err)
		}
	}

	return signedIn, nil
}

func SignInWithFacebook(ctx context.Context, h *handler.Handler, userID string, accessToken, email string) (bool, error) {
	config := h.Config(ctx)

	if !config.FacebookSignInEnabled {
		return false, errors.New("Facebook sign in is disabled")
	}
	if config.FacebookSignInAppID == "" {
		return false, errors.New("Facebook sign in app id is empty")
	}
	if config.FacebookSignInAppSecret == "" {
		return false, errors.New("Facebook sign in app secret is empty")
	}
	if strings.TrimSpace(email) == "" {
		return false, errors.New("Facebook sign in email is empty")
	}

	endpoint := fmt.Sprintf(
		"https://graph.facebook.com/debug_token?input_token=%v&access_token=%v|%v",
		accessToken,
		config.FacebookSignInAppID,
		config.FacebookSignInAppSecret,
	)
	res, err := client.Get(endpoint)
	if err != nil {
		return false, fmt.Errorf("fetch Facebook access token debug data: %w", err)
	}
	defer res.Body.Close()

	var token struct {
		Data struct {
			AppID   string `json:"app_id"`
			UserID  string `json:"user_id"`
			IsValid bool   `json:"is_valid"`
		}
	}
	if err := httpx.RelaxedDecodeJSON(&token, res.Body); err != nil {
		return false, fmt.Errorf("decode Facebook access token debug data: %w", err)
	}

	if !token.Data.IsValid {
		return false, errors.New("invalid access token")
	}
	if token.Data.AppID != config.FacebookSignInAppID {
		return false, errors.New("app id from access token inspection does not match the one configured")
	}
	if token.Data.UserID != userID {
		return false, errors.New("user id from access token inspection does not match the one given")
	}

	endpoint = fmt.Sprintf("https://graph.facebook.com/v18.0/me?access_token=%v&fields=email", accessToken)
	res, err = client.Get(endpoint)
	if err != nil {
		return false, fmt.Errorf("fetch Facebook user data: %w", err)
	}
	defer res.Body.Close()

	var me struct {
		Email string
	}
	if err := httpx.RelaxedDecodeJSON(&me, res.Body); err != nil {
		return false, fmt.Errorf("decode Facebook user data: %w", err)
	}

	if me.Email != email {
		return false, errors.New("user email from Facebook does not match the one given")
	}

	behaviour := account.FacebookSignInOnly
	if config.SignUpEnabled {
		if config.SignUpAutoActivateEnabled {
			behaviour = account.FacebookAllowSignUpActivate
		} else {
			behaviour = account.FacebookAllowSignUp
		}
	}

	signedIn, err := h.Svc.Account.SignInWithFacebook(ctx, me.Email, behaviour)
	if err != nil {
		return false, fmt.Errorf("sign in wih Facebook: %w", err)
	}

	if signedIn {
		if _, err := h.RenewSession(ctx); err != nil {
			return signedIn, fmt.Errorf("renew session: %w", err)
		}

		if err := SignInSetSession(ctx, h, me.Email); err != nil {
			return signedIn, fmt.Errorf("sign in set session: %w", err)
		}
	}

	return signedIn, nil
}

func SignInSetSession(ctx context.Context, h *handler.Handler, email string) error {
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
