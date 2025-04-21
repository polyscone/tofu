package handler

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/polyscone/tofu/internal/session"
)

const (
	// Global session keys
	skeyLastView       = "global.last_view"
	skeyFlash          = "global.flash"
	skeyFlashWarning   = "global.flash_warning"
	skeyFlashImportant = "global.flash_important"
	skeyFlashError     = "global.flash_error"
	skeyRedirect       = "global.redirect"
	skeySortTopID      = "global.sort_top_id"
	skeyHighlightID    = "global.highlight_id"
	skeyURLValues      = "global.url_values"

	// Account session keys
	skeyUserID                   = "account.user_id"
	skeyImposterUserID           = "account.imposter_user_id"
	skeyEmail                    = "account.email"
	skeyTOTPMethod               = "account.totp_method"
	skeyHasActivatedTOTP         = "account.has_verified_totp"
	skeyIsAwaitingTOTP           = "account.is_awaiting_totp"
	skeyIsSignedIn               = "account.is_signed_in"
	skeyKnownPasswordBreachCount = "account.password_known_breach_count"
	skeySignInAttempts           = "account.sign_in_attempts"
	skeyLastSignInAttemptAt      = "account.last_sign_in_attempt_at"
)

type Session struct {
	*session.Manager
}

func (s *Session) LastView(ctx context.Context) string {
	return s.GetString(ctx, skeyLastView)
}

func (s *Session) SetLastView(ctx context.Context, value string) {
	s.Set(ctx, skeyLastView, value)
}

func (s *Session) PopLastView(ctx context.Context) string {
	return s.PopString(ctx, skeyLastView)
}

func (s *Session) Flash(ctx context.Context) []string {
	return s.GetStrings(ctx, skeyFlash)
}

func (s *Session) SetFlash(ctx context.Context, value []string) {
	s.Set(ctx, skeyFlash, value)
}

func (s *Session) PopFlash(ctx context.Context) []string {
	return s.PopStrings(ctx, skeyFlash)
}

func (s *Session) DeleteFlash(ctx context.Context) {
	s.Delete(ctx, skeyFlash)
}

func (s *Session) FlashWarning(ctx context.Context) []string {
	return s.GetStrings(ctx, skeyFlashWarning)
}

func (s *Session) SetFlashWarning(ctx context.Context, value []string) {
	s.Set(ctx, skeyFlashWarning, value)
}

func (s *Session) PopFlashWarning(ctx context.Context) []string {
	return s.PopStrings(ctx, skeyFlashWarning)
}

func (s *Session) DeleteFlashWarning(ctx context.Context) {
	s.Delete(ctx, skeyFlashWarning)
}

func (s *Session) FlashImportant(ctx context.Context) []string {
	return s.GetStrings(ctx, skeyFlashImportant)
}

func (s *Session) SetFlashImportant(ctx context.Context, value []string) {
	s.Set(ctx, skeyFlashImportant, value)
}

func (s *Session) PopFlashImportant(ctx context.Context) []string {
	return s.PopStrings(ctx, skeyFlashImportant)
}

func (s *Session) DeleteFlashImportant(ctx context.Context) {
	s.Delete(ctx, skeyFlashImportant)
}

func (s *Session) FlashError(ctx context.Context) []string {
	return s.GetStrings(ctx, skeyFlashError)
}

func (s *Session) SetFlashError(ctx context.Context, value []string) {
	s.Set(ctx, skeyFlashError, value)
}

func (s *Session) PopFlashError(ctx context.Context) []string {
	return s.PopStrings(ctx, skeyFlashError)
}

func (s *Session) DeleteFlashError(ctx context.Context) {
	s.Delete(ctx, skeyFlashError)
}

func (s *Session) Redirect(ctx context.Context) string {
	return s.GetString(ctx, skeyRedirect)
}

func (s *Session) SetRedirect(ctx context.Context, value string) {
	s.Set(ctx, skeyRedirect, value)
}

func (s *Session) PopRedirect(ctx context.Context) string {
	return s.PopString(ctx, skeyRedirect)
}

func (s *Session) DeleteRedirect(ctx context.Context) {
	s.Delete(ctx, skeyRedirect)
}

func (s *Session) SortTopID(ctx context.Context) int {
	return s.GetInt(ctx, skeySortTopID)
}

func (s *Session) SetSortTopID(ctx context.Context, value int) {
	s.Set(ctx, skeySortTopID, value)
}

func (s *Session) PopSortTopID(ctx context.Context) int {
	return s.PopInt(ctx, skeySortTopID)
}

func (s *Session) DeleteSortTopID(ctx context.Context) {
	s.Delete(ctx, skeySortTopID)
}

func (s *Session) HighlightID(ctx context.Context) int {
	return s.GetInt(ctx, skeyHighlightID)
}

func (s *Session) SetHighlightID(ctx context.Context, value int) {
	s.Set(ctx, skeyHighlightID, value)
}

func (s *Session) PopHighlightID(ctx context.Context) int {
	return s.PopInt(ctx, skeyHighlightID)
}

func (s *Session) URLValues(ctx context.Context) url.Values {
	var values url.Values
	data := []byte(s.GetString(ctx, skeyURLValues))
	json.Unmarshal(data, &values)

	return values
}

func (s *Session) SetURLValues(ctx context.Context, value url.Values) {
	b, _ := json.Marshal(value)

	s.Set(ctx, skeyURLValues, string(b))
}

func (s *Session) PopURLValues(ctx context.Context) url.Values {
	var values url.Values
	data := []byte(s.PopString(ctx, skeyURLValues))
	json.Unmarshal(data, &values)

	return values
}

func (s *Session) DeleteHighlightID(ctx context.Context) {
	s.Delete(ctx, skeyHighlightID)
}

func (s *Session) ImposterUserID(ctx context.Context) int {
	return s.GetInt(ctx, skeyImposterUserID)
}

func (s *Session) SetImposterUserID(ctx context.Context, value int) {
	s.Set(ctx, skeyImposterUserID, value)
}

func (s *Session) PopImposterUserID(ctx context.Context) int {
	return s.PopInt(ctx, skeyImposterUserID)
}

func (s *Session) DeleteImposterUserID(ctx context.Context) {
	s.Delete(ctx, skeyImposterUserID)
}

func (s *Session) UserID(ctx context.Context) int {
	return s.GetInt(ctx, skeyUserID)
}

func (s *Session) SetUserID(ctx context.Context, value int) {
	s.Set(ctx, skeyUserID, value)
}

func (s *Session) PopUserID(ctx context.Context) int {
	return s.PopInt(ctx, skeyUserID)
}

func (s *Session) DeleteUserID(ctx context.Context) {
	s.Delete(ctx, skeyUserID)
}

func (s *Session) Email(ctx context.Context) string {
	return s.GetString(ctx, skeyEmail)
}

func (s *Session) SetEmail(ctx context.Context, value string) {
	s.Set(ctx, skeyEmail, value)
}

func (s *Session) PopEmail(ctx context.Context) string {
	return s.PopString(ctx, skeyEmail)
}

func (s *Session) DeleteEmail(ctx context.Context) {
	s.Delete(ctx, skeyEmail)
}

func (s *Session) TOTPMethod(ctx context.Context) string {
	return s.GetString(ctx, skeyTOTPMethod)
}

func (s *Session) SetTOTPMethod(ctx context.Context, value string) {
	s.Set(ctx, skeyTOTPMethod, value)
}

func (s *Session) PopTOTPMethod(ctx context.Context) string {
	return s.PopString(ctx, skeyTOTPMethod)
}

func (s *Session) DeleteTOTPMethod(ctx context.Context) {
	s.Delete(ctx, skeyTOTPMethod)
}

func (s *Session) HasActivatedTOTP(ctx context.Context) bool {
	return s.GetBool(ctx, skeyHasActivatedTOTP)
}

func (s *Session) SetHasActivatedTOTP(ctx context.Context, value bool) {
	s.Set(ctx, skeyHasActivatedTOTP, value)
}

func (s *Session) PopHasActivatedTOTP(ctx context.Context) bool {
	return s.PopBool(ctx, skeyHasActivatedTOTP)
}

func (s *Session) DeleteHasActivatedTOTP(ctx context.Context) {
	s.Delete(ctx, skeyHasActivatedTOTP)
}

func (s *Session) IsAwaitingTOTP(ctx context.Context) bool {
	return s.GetBool(ctx, skeyIsAwaitingTOTP)
}

func (s *Session) SetIsAwaitingTOTP(ctx context.Context, value bool) {
	s.Set(ctx, skeyIsAwaitingTOTP, value)
}

func (s *Session) PopIsAwaitingTOTP(ctx context.Context) bool {
	return s.PopBool(ctx, skeyIsAwaitingTOTP)
}

func (s *Session) DeleteIsAwaitingTOTP(ctx context.Context) {
	s.Delete(ctx, skeyIsAwaitingTOTP)
}

func (s *Session) IsSignedIn(ctx context.Context) bool {
	return s.GetBool(ctx, skeyIsSignedIn)
}

func (s *Session) SetIsSignedIn(ctx context.Context, value bool) {
	s.Set(ctx, skeyIsSignedIn, value)
}

func (s *Session) PopIsSignedIn(ctx context.Context) bool {
	return s.PopBool(ctx, skeyIsSignedIn)
}

func (s *Session) DeleteIsSignedIn(ctx context.Context) {
	s.Delete(ctx, skeyIsSignedIn)
}

func (s *Session) KnownPasswordBreachCount(ctx context.Context) int {
	return s.GetInt(ctx, skeyKnownPasswordBreachCount)
}

func (s *Session) SetKnownPasswordBreachCount(ctx context.Context, value int) {
	s.Set(ctx, skeyKnownPasswordBreachCount, value)
}

func (s *Session) PopKnownPasswordBreachCount(ctx context.Context) int {
	return s.PopInt(ctx, skeyKnownPasswordBreachCount)
}

func (s *Session) DeleteKnownPasswordBreachCount(ctx context.Context) {
	s.Delete(ctx, skeyKnownPasswordBreachCount)
}

func (s *Session) SignInAttempts(ctx context.Context) int {
	return s.GetInt(ctx, skeySignInAttempts)
}

func (s *Session) SetSignInAttempts(ctx context.Context, value int) {
	s.Set(ctx, skeySignInAttempts, value)
}

func (s *Session) PopSignInAttempts(ctx context.Context) int {
	return s.PopInt(ctx, skeySignInAttempts)
}

func (s *Session) DeleteSignInAttempts(ctx context.Context) {
	s.Delete(ctx, skeySignInAttempts)
}

func (s *Session) LastSignInAttemptAt(ctx context.Context) time.Time {
	return s.GetTime(ctx, skeyLastSignInAttemptAt)
}

func (s *Session) SetLastSignInAttemptAt(ctx context.Context, value time.Time) {
	s.Set(ctx, skeyLastSignInAttemptAt, value)
}

func (s *Session) PopLastSignInAttemptAt(ctx context.Context) time.Time {
	return s.PopTime(ctx, skeyLastSignInAttemptAt)
}

func (s *Session) DeleteLastSignInAttemptAt(ctx context.Context) {
	s.Delete(ctx, skeyLastSignInAttemptAt)
}
