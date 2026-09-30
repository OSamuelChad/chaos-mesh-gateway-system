package chaos

import "github.com/gofiber/fiber/v2"

type latencyRequest struct {
	LatencyMs *int64 `json:"latency_ms"`
}

type faultsRequest struct {
	Rate *int64 `json:"rate"`
}

// RegisterRoutes registra as rotas administrativas de caos.
//
//	POST /chaos/latency {"latency_ms": 2000}
//	POST /chaos/faults  {"rate": 80}
//	POST /chaos/reset
//	GET  /chaos/status
func RegisterRoutes(router fiber.Router, c *Controller) {
	g := router.Group("/chaos")

	g.Get("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(c.Snapshot())
	})

	g.Post("/latency", func(ctx *fiber.Ctx) error {
		var req latencyRequest
		if err := ctx.BodyParser(&req); err != nil || req.LatencyMs == nil {
			return fiber.NewError(fiber.StatusBadRequest, `corpo inválido: esperado {"latency_ms": <0-10000>}`)
		}
		if err := c.SetLatency(*req.LatencyMs); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return ctx.JSON(c.Snapshot())
	})

	g.Post("/faults", func(ctx *fiber.Ctx) error {
		var req faultsRequest
		if err := ctx.BodyParser(&req); err != nil || req.Rate == nil {
			return fiber.NewError(fiber.StatusBadRequest, `corpo inválido: esperado {"rate": <0-100>}`)
		}
		if err := c.SetFaultRate(*req.Rate); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return ctx.JSON(c.Snapshot())
	})

	g.Post("/reset", func(ctx *fiber.Ctx) error {
		c.Reset()
		return ctx.JSON(c.Snapshot())
	})
}
