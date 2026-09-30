// service-payment: motor fictício de processamento de pagamentos com
// middleware de caos programável.
package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"service-payment/chaos"
)

func main() {
	port := getenv("PORT", "8081")
	controller := chaos.NewController()

	app := fiber.New(fiber.Config{
		AppName:      "service-payment",
		ErrorHandler: jsonErrorHandler,
	})
	app.Use(recover.New(), logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"service": "payment", "health": controller.Snapshot().Health})
	})

	// Rotas administrativas de caos (acessíveis apenas pela rede interna).
	chaos.RegisterRoutes(app, controller)

	// Rota de negócio protegida pelo middleware de caos.
	app.Post("/process", controller.Middleware(), processHandler)

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("falha ao iniciar servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("encerrando service-payment...")
	if err := app.Shutdown(); err != nil {
		log.Printf("erro no shutdown: %v", err)
	}
}

func jsonErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
	}
	msg := err.Error()
	if code == fiber.StatusInternalServerError && fe == nil {
		log.Printf("erro interno: %v", err)
		msg = "erro interno"
	}
	return c.Status(code).JSON(fiber.Map{"error": msg, "status": code})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
