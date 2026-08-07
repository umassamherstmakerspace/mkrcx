package leash_signin

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func TestLoginReturnPolicy(t *testing.T) {
	policy, err := newLoginReturnPolicy("https://mkr.cx, https://staging.mkr.cx, http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}

	allowed := []string{
		"/login",
		"https://mkr.cx/login",
		"https://staging.mkr.cx/login",
		"http://localhost:5173/login",
	}
	for _, candidate := range allowed {
		if _, err := policy.validate(candidate); err != nil {
			t.Errorf("expected %q to be allowed: %v", candidate, err)
		}
	}

	rejected := []string{
		"https://attacker.example/collect",
		"//attacker.example/collect",
		"javascript:alert(1)",
		"https://user:password@mkr.cx/login",
		"https://mkr.cx/login?next=https://attacker.example",
	}
	for _, candidate := range rejected {
		if _, err := policy.validate(candidate); err == nil {
			t.Errorf("expected %q to be rejected", candidate)
		}
	}
}

func TestLoginCodeIsSingleUseAndExpires(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:login-code-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Session{}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	code, hash, err := newLoginCode()
	if err != nil {
		t.Fatal(err)
	}
	codeExpiresAt := now.Add(time.Minute)
	session := models.Session{
		SessionID:          "valid-session",
		UserID:             1,
		ExpiresAt:          now.Add(time.Hour),
		LoginCodeHash:      &hash,
		LoginCodeExpiresAt: &codeExpiresAt,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}

	claimed, err := claimLoginCode(db, code, now)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.SessionID != session.SessionID || claimed.LoginCodeConsumedAt == nil {
		t.Fatalf("unexpected claimed session: %+v", claimed)
	}
	if _, err := claimLoginCode(db, code, now); !errors.Is(err, errInvalidLoginCode) {
		t.Fatalf("replayed code error = %v, want %v", err, errInvalidLoginCode)
	}

	expiredCode, expiredHash, err := newLoginCode()
	if err != nil {
		t.Fatal(err)
	}
	expiredAt := now.Add(-time.Second)
	expiredSession := models.Session{
		SessionID:          "expired-session",
		UserID:             1,
		ExpiresAt:          now.Add(time.Hour),
		LoginCodeHash:      &expiredHash,
		LoginCodeExpiresAt: &expiredAt,
	}
	if err := db.Create(&expiredSession).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := claimLoginCode(db, expiredCode, now); !errors.Is(err, errInvalidLoginCode) {
		t.Fatalf("expired code error = %v, want %v", err, errInvalidLoginCode)
	}
}

func TestLoginNonceIsRandomAndBrowserBound(t *testing.T) {
	nonce, hash, err := newLoginNonce()
	if err != nil {
		t.Fatal(err)
	}
	if !validLoginNonce(nonce, hash) {
		t.Fatal("fresh login nonce did not validate")
	}
	other, _, err := newLoginNonce()
	if err != nil {
		t.Fatal(err)
	}
	if validLoginNonce(other, hash) || validLoginNonce("malformed", hash) {
		t.Fatal("unrelated or malformed login nonce validated")
	}
}

func TestLoginRedirectContainsOnlyOneTimeCode(t *testing.T) {
	redirect, err := loginRedirect("https://staging.mkr.cx/login", "one-time-code", "opaque-state")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("code") != "one-time-code" || parsed.Query().Get("state") != "opaque-state" {
		t.Fatalf("unexpected redirect query: %s", parsed.RawQuery)
	}
	if parsed.Query().Has("token") || parsed.Query().Has("expires_at") {
		t.Fatalf("redirect leaked session material: %s", parsed.RawQuery)
	}
}
