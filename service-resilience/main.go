// service-resilience: envolve o service-payment com um Circuit Breaker.
package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"service-resilience/breaker"
)

func main() {
	port := getenv("PORT", "8082")
	engine := breaker.NewEngine(breaker.DefaultConfig(getenv("PAYMENT_URL", "http://localhost:8081")), nil)

	app := fiber.New(fiber.Config{
		AppName:      "service-resilience",
		ErrorHandler: jsonErrorHandler,
	})
	app.Use(recover.New(), logger.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"service": "resilience", "state": engine.State()})
	})

	r := app.Group("/resilience")

	// Métricas e estado atual do Circuit Breaker.
	r.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(engine.Status())
	})

	// Processa uma cobrança através do Circuit Breaker.
	r.Post("/process", func(c *fiber.Ctx) error {
		out := engine.Process(c.UserContext(), c.Body())
		if out.RetryAfter > 0 {
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(out.RetryAfter))
		}
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		return c.Status(out.StatusCode).Send(out.Body)
	})

	// Força o breaker de volta para CLOSED.
	r.Post("/reset", func(c *fiber.Ctx) error {
		engine.Reset()
		return c.JSON(engine.Status())
	})

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("falha ao iniciar servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("encerrando service-resilience...")
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
