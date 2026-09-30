package hateoas

import "github.com/gofiber/fiber/v2"

// ErrorResponse é o corpo padrão de erro — também carrega "_links".
type ErrorResponse struct {
	Error  string `json:"error" example:"token ausente ou inválido"`
	Status int    `json:"status" example:"401"`
	Resource
}

// Error responde com um erro JSON contendo links de navegação.
func Error(c *fiber.Ctx, status int, message string, links Links) error {
	return c.Status(status).JSON(ErrorResponse{Error: message, Status: status, Resource: Resource{Links: links}})
}
