// Package handlers implementa as rotas do API Gateway.
package handlers

import (
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
)

// Version é a versão da API.
const Version = "1.0.0"

// Handler agrupa as dependências das rotas protegidas.
type Handler struct {
	up        *Upstream
	startedAt time.Time
}

// New cria o handler.
func New(up *Upstream) *Handler {
	return &Handler{up: up, startedAt: time.Now()}
}

// upstreamFailure converte um erro de serviço interno em 502 com links de navegação.
func upstreamFailure(c *fiber.Ctx, service string, err error, links hateoas.Links) error {
	var ue *UpstreamError
	if errors.As(err, &ue) && ue.Status < fiber.StatusInternalServerError {
		return hateoas.Error(c, ue.Status, ue.Body, links)
	}
	log.Printf("falha ao chamar %s: %v", service, err)
	return hateoas.Error(c, fiber.StatusBadGateway, service+" indisponível", links)
}
