package handlers

import (
	"crypto/subtle"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
	"api-gateway/middleware"
)

// LoginRequest são as credenciais de acesso.
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"password"`
}

// LoginResponse contém o token JWT emitido.
type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time `json:"expires_at"`
	ExpiresIn   int64     `json:"expires_in" example:"3600"`
	hateoas.Resource
}

// AuthHandler emite tokens para credenciais válidas.
type AuthHandler struct {
	tokens   *middleware.TokenIssuer
	username []byte
	password []byte
}

// NewAuthHandler cria o handler de autenticação.
func NewAuthHandler(tokens *middleware.TokenIssuer, username, password string) *AuthHandler {
	return &AuthHandler{tokens: tokens, username: []byte(username), password: []byte(password)}
}

// Login godoc
//
//	@Summary		Autenticar
//	@Description	Troca usuário e senha por um token JWT (HS256) válido por 1 hora.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		LoginRequest	true	"Credenciais"
//	@Success		200			{object}	LoginResponse
//	@Failure		400			{object}	hateoas.ErrorResponse
//	@Failure		401			{object}	hateoas.ErrorResponse
//	@Failure		429			{object}	hateoas.ErrorResponse
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return hateoas.Error(c, fiber.StatusBadRequest, "corpo JSON inválido", hateoas.AnonymousLinks())
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" || len(username) > 64 || len(req.Password) > 128 {
		return hateoas.Error(c, fiber.StatusBadRequest, "username e password são obrigatórios", hateoas.AnonymousLinks())
	}

	userOK := subtle.ConstantTimeCompare([]byte(username), h.username) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), h.password) == 1
	if !userOK || !passOK {
		return hateoas.Error(c, fiber.StatusUnauthorized, "credenciais inválidas", hateoas.AnonymousLinks())
	}

	token, expiresAt, err := h.tokens.Issue(username)
	if err != nil {
		return hateoas.Error(c, fiber.StatusInternalServerError, "falha ao emitir token", hateoas.AnonymousLinks())
	}

	return c.JSON(LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
		Resource:    hateoas.Resource{Links: hateoas.RootLinks()},
	})
}
