package chaos

import "github.com/gofiber/fiber/v2"

// Middleware aplica latência (time.Sleep) e falhas HTTP 500 nas rotas que envolve.
func (c *Controller) Middleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if c.apply() {
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":  "falha de infraestrutura simulada pelo middleware de caos",
				"status": fiber.StatusInternalServerError,
				"chaos":  true,
			})
		}
		return ctx.Next()
	}
}
