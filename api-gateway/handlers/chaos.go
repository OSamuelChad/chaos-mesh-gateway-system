package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
)

const (
	maxLatencyMs = 10_000
	maxFaultRate = 100
)

// SetLatency godoc
//
//	@Summary		Injetar latência no Payment Service
//	@Description	Define a latência artificial (0-10000 ms) aplicada pelo middleware de caos.
//	@Tags			chaos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		LatencyRequest	true	"Latência em ms"
//	@Success		200		{object}	NodeResponse
//	@Failure		400		{object}	hateoas.ErrorResponse
//	@Failure		401		{object}	hateoas.ErrorResponse
//	@Failure		502		{object}	hateoas.ErrorResponse
//	@Router			/api/v1/services/payment/chaos/latency [put]
func (h *Handler) SetLatency(c *fiber.Ctx) error {
	var req LatencyRequest
	if err := c.BodyParser(&req); err != nil || req.LatencyMs == nil || *req.LatencyMs < 0 || *req.LatencyMs > maxLatencyMs {
		return hateoas.Error(c, fiber.StatusBadRequest, `esperado {"latency_ms": <0-10000>}`, h.paymentLinks(c))
	}
	return h.paymentChaos(c, "/chaos/latency", map[string]int64{"latency_ms": *req.LatencyMs})
}

// SetFaults godoc
//
//	@Summary		Injetar taxa de erros no Payment Service
//	@Description	Define a porcentagem (0-100) de requisições que falharão com HTTP 500.
//	@Tags			chaos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		FaultsRequest	true	"Taxa de erros (%)"
//	@Success		200		{object}	NodeResponse
//	@Failure		400		{object}	hateoas.ErrorResponse
//	@Failure		401		{object}	hateoas.ErrorResponse
//	@Failure		502		{object}	hateoas.ErrorResponse
//	@Router			/api/v1/services/payment/chaos/faults [put]
func (h *Handler) SetFaults(c *fiber.Ctx) error {
	var req FaultsRequest
	if err := c.BodyParser(&req); err != nil || req.Rate == nil || *req.Rate < 0 || *req.Rate > maxFaultRate {
		return hateoas.Error(c, fiber.StatusBadRequest, `esperado {"rate": <0-100>}`, h.paymentLinks(c))
	}
	return h.paymentChaos(c, "/chaos/faults", map[string]int64{"rate": *req.Rate})
}

// Heal godoc
//
//	@Summary	Restaurar o Payment Service
//	@Tags		chaos
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	NodeResponse
//	@Failure	401	{object}	hateoas.ErrorResponse
//	@Failure	502	{object}	hateoas.ErrorResponse
//	@Router		/api/v1/services/payment/chaos/reset [post]
func (h *Handler) Heal(c *fiber.Ctx) error {
	return h.paymentChaos(c, "/chaos/reset", nil)
}

// ResetBreaker godoc
//
//	@Summary	Fechar o Circuit Breaker
//	@Tags		chaos
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	NodeResponse
//	@Failure	401	{object}	hateoas.ErrorResponse
//	@Failure	502	{object}	hateoas.ErrorResponse
//	@Router		/api/v1/services/resilience/breaker/reset [post]
func (h *Handler) ResetBreaker(c *fiber.Ctx) error {
	if err := h.up.callJSON(c.UserContext(), http.MethodPost, h.up.resilienceURL+"/resilience/reset", nil, nil); err != nil {
		return upstreamFailure(c, "resilience service", err, hateoas.ForResilience(h.up.snapshot(c.UserContext()).state()))
	}
	return c.JSON(h.resilienceNode(c))
}

func (h *Handler) paymentChaos(c *fiber.Ctx, path string, body interface{}) error {
	if body == nil {
		body = struct{}{}
	}
	if err := h.up.callJSON(c.UserContext(), http.MethodPost, h.up.paymentURL+path, body, nil); err != nil {
		return upstreamFailure(c, "payment service", err, h.paymentLinks(c))
	}
	return c.JSON(h.paymentNode(c))
}

func (h *Handler) paymentLinks(c *fiber.Ctx) hateoas.Links {
	return hateoas.ForPayment(h.up.snapshot(c.UserContext()).state())
}
