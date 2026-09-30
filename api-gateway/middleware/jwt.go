// Package middleware contém a emissão e a verificação de tokens JWT.
package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"api-gateway/hateoas"
)

const (
	// LocalUser é a chave em c.Locals com o usuário autenticado.
	LocalUser = "user"
	issuer    = "chaos-mesh-gateway"
)

// MinSecretLength é o tamanho mínimo aceito para o segredo HMAC.
const MinSecretLength = 32

var ErrShortSecret = errors.New("JWT_SECRET deve ter pelo menos 32 bytes")

// Claims são as claims do token emitido pelo gateway.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// TokenIssuer emite e valida tokens HS256.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenIssuer valida o segredo e cria o emissor.
func NewTokenIssuer(secret []byte, ttl time.Duration) (*TokenIssuer, error) {
	if len(secret) < MinSecretLength {
		return nil, ErrShortSecret
	}
	return &TokenIssuer{secret: secret, ttl: ttl, now: time.Now}, nil
}

// Issue gera um token assinado para o usuário.
func (t *TokenIssuer) Issue(subject string) (string, time.Time, error) {
	now := t.now()
	expiresAt := now.Add(t.ttl)
	claims := Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, expiresAt, err
}

// Parse valida assinatura, algoritmo, emissor e expiração.
func (t *TokenIssuer) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (interface{}, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// RequireJWT exige o header "Authorization: Bearer <token>".
func RequireJWT(t *TokenIssuer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		scheme, token, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			return hateoas.Error(c, fiber.StatusUnauthorized, "token Bearer ausente", hateoas.AnonymousLinks())
		}

		claims, err := t.Parse(strings.TrimSpace(token))
		if err != nil {
			return hateoas.Error(c, fiber.StatusUnauthorized, "token inválido ou expirado", hateoas.AnonymousLinks())
		}

		c.Locals(LocalUser, claims.Subject)
		return c.Next()
	}
}
