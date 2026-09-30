package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
)

// ProcessPayment godoc
//
//	@Summary		Processar cobrança
//	@Description	Encaminha a cobrança ao Resilience Service, que a envia ao Payment Service através do Circuit Breaker. Com o breaker OPEN retorna 503 com payload de fallback.
//	@Tags			payments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payment	body		object	true	"Cobrança"	example({"amount":149.9,"currency":"BRL","customer":"cliente-demo"})
//	@Success		201		{object}	PaymentResult
//	@Failure		400		{object}	PaymentResult
//	@Failure		401		{object}	hateoas.ErrorResponse
//	@Failure		502		{object}	PaymentResult
//	@Failure		503		{object}	PaymentResult
//	@Router			/api/v1/payments [post]
func (h *Handler) ProcessPayment(c *fiber.Ctx) error {
	if !json.Valid(c.Body()) {
		return hateoas.Error(c, fiber.StatusBadRequest, "corpo JSON inválido", h.paymentLinks(c))
	}

	reply, err := h.up.call(c.UserContext(), http.MethodPost, h.up.resilienceURL+"/resilience/process", c.Body())
	if err != nil {
		return upstreamFailure(c, "resilience service", err, h.paymentLinks(c))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(reply.body, &result); err != nil {
		result = map[string]interface{}{"raw": string(reply.body)}
	}
	fallback, _ := result["fallback"].(bool)
	if reply.retryAfter != "" {
		c.Set(fiber.HeaderRetryAfter, reply.retryAfter)
	}

	return c.Status(reply.status).JSON(PaymentResult{
		StatusCode: reply.status,
		Fallback:   fallback,
		Result:     result,
		Resource:   hateoas.Resource{Links: h.paymentLinks(c)},
	})
}
