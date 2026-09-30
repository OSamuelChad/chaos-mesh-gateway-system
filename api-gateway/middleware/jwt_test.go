package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func newApp(t *testing.T, issuer *TokenIssuer) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Get("/protected", RequireJWT(issuer), func(c *fiber.Ctx) error {
		return c.SendString(c.Locals(LocalUser).(string))
	})
	return app
}

func status(t *testing.T, app *fiber.App, auth string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/protected", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func TestRejectsShortSecret(t *testing.T) {
	if _, err := NewTokenIssuer([]byte("curto"), time.Hour); err == nil {
		t.Fatal("esperava erro para segredo curto")
	}
}

func TestValidTokenIsAccepted(t *testing.T) {
	issuer, _ := NewTokenIssuer(testSecret, time.Hour)
	token, _, err := issuer.Issue("admin")
	if err != nil {
		t.Fatal(err)
	}
	if got := status(t, newApp(t, issuer), "Bearer "+token); got != 200 {
		t.Fatalf("status = %d, want 200", got)
	}
}

func TestMissingOrMalformedTokenIsRejected(t *testing.T) {
	issuer, _ := NewTokenIssuer(testSecret, time.Hour)
	app := newApp(t, issuer)
	for _, auth := range []string{"", "Bearer", "Basic abc", "Bearer nao.e.jwt"} {
		if got := status(t, app, auth); got != 401 {
			t.Errorf("auth %q: status = %d, want 401", auth, got)
		}
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	issuer, _ := NewTokenIssuer(testSecret, time.Minute)
	issuer.now = func() time.Time { return time.Now().Add(-2 * time.Minute) }
	token, _, _ := issuer.Issue("admin")
	issuer.now = time.Now
	if got := status(t, newApp(t, issuer), "Bearer "+token); got != 401 {
		t.Fatalf("status = %d, want 401", got)
	}
}

func TestTokenSignedWithOtherSecretIsRejected(t *testing.T) {
	other, _ := NewTokenIssuer([]byte("ffffffffffffffffffffffffffffffff"), time.Hour)
	token, _, _ := other.Issue("admin")
	issuer, _ := NewTokenIssuer(testSecret, time.Hour)
	if got := status(t, newApp(t, issuer), "Bearer "+token); got != 401 {
		t.Fatalf("status = %d, want 401", got)
	}
}

func TestNoneAlgorithmIsRejected(t *testing.T) {
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject: "admin", Issuer: issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	iss, _ := NewTokenIssuer(testSecret, time.Hour)
	if got := status(t, newApp(t, iss), "Bearer "+token); got != 401 {
		t.Fatalf("status = %d, want 401", got)
	}
}
