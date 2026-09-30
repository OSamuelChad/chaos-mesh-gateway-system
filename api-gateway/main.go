// API Gateway da Chaos Mesh: autenticação JWT, proxy/agregador dos
// microsserviços internos, documentação Swagger e motor HATEOAS.
//
//	@title						Chaos Mesh Gateway API
//	@version					1.0
//	@description				Gateway da malha de resiliência. Todas as respostas trazem "_links" (HATEOAS) calculados a partir da saúde dos serviços e do estado do Circuit Breaker.
//	@BasePath					/
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Informe "Bearer {token}" obtido em /api/v1/auth/login.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"

	_ "api-gateway/docs"
	"api-gateway/handlers"
	"api-gateway/hateoas"
	"api-gateway/middleware"
)

const tokenTTL = time.Hour

type config struct {
	port          string
	jwtSecret     []byte
	adminUser     string
	adminPassword string
	paymentURL    string
	resilienceURL string
	corsOrigins   string
}

func loadConfig() config {
	cfg := config{
		port:          getenv("PORT", "8080"),
		jwtSecret:     []byte(os.Getenv("JWT_SECRET")),
		adminUser:     getenv("ADMIN_USERNAME", "admin"),
		adminPassword: getenv("ADMIN_PASSWORD", "password"),
		paymentURL:    getenv("PAYMENT_URL", "http://localhost:8081"),
		resilienceURL: getenv("RESILIENCE_URL", "http://localhost:8082"),
		corsOrigins:   getenv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"),
	}
	if len(cfg.jwtSecret) == 0 {
		cfg.jwtSecret = randomSecret()
		log.Println("AVISO: JWT_SECRET não definido; usando segredo aleatório (tokens expiram ao reiniciar).")
	}
	if cfg.adminPassword == "password" {
		log.Println("AVISO: usando credenciais padrão admin/password; defina ADMIN_PASSWORD fora do ambiente de demonstração.")
	}
	return cfg
}

func main() {
	cfg := loadConfig()

	tokens, err := middleware.NewTokenIssuer(cfg.jwtSecret, tokenTTL)
	if err != nil {
		log.Fatalf("configuração inválida: %v", err)
	}

	app := fiber.New(fiber.Config{AppName: "api-gateway", ErrorHandler: errorHandler})
	app.Use(recover.New(), logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.corsOrigins,
		AllowHeaders: "Authorization, Content-Type",
		AllowMethods: "GET, POST, PUT, OPTIONS",
	}))

	app.Get("/swagger/*", swagger.HandlerDefault)
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"service": "api-gateway", "status": "UP"}) })

	registerRoutes(app, cfg, tokens)

	go func() {
		if err := app.Listen(":" + cfg.port); err != nil {
			log.Fatalf("falha ao iniciar servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("encerrando api-gateway...")
	if err := app.Shutdown(); err != nil {
		log.Printf("erro no shutdown: %v", err)
	}
}

func registerRoutes(app *fiber.App, cfg config, tokens *middleware.TokenIssuer) {
	auth := handlers.NewAuthHandler(tokens, cfg.adminUser, cfg.adminPassword)
	h := handlers.New(handlers.NewUpstream(cfg.paymentURL, cfg.resilienceURL))

	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return hateoas.Error(c, fiber.StatusTooManyRequests, "muitas tentativas de login; aguarde 1 minuto", hateoas.AnonymousLinks())
		},
	})

	// Pública
	app.Post(hateoas.PathLogin, loginLimiter, auth.Login)

	// Protegidas por JWT
	jwt := middleware.RequireJWT(tokens)
	app.Get(hateoas.PathRoot, jwt, h.Root)
	app.Get(hateoas.PathTopology, jwt, h.Topology)
	app.Get(hateoas.PathGateway, jwt, h.GatewayNode)
	app.Get(hateoas.PathPayment, jwt, h.PaymentNode)
	app.Put(hateoas.PathPaymentLatency, jwt, h.SetLatency)
	app.Put(hateoas.PathPaymentFaults, jwt, h.SetFaults)
	app.Post(hateoas.PathPaymentHeal, jwt, h.Heal)
	app.Get(hateoas.PathResilience, jwt, h.ResilienceNode)
	app.Post(hateoas.PathResilienceReset, jwt, h.ResetBreaker)
	app.Post(hateoas.PathPayments, jwt, h.ProcessPayment)
	app.Post(hateoas.PathTraffic, jwt, h.SimulateTraffic)
}

// errorHandler garante que até os erros do framework (404, 405...) tragam "_links".
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "erro interno"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	} else {
		log.Printf("erro interno: %v", err)
	}
	return hateoas.Error(c, code, msg, hateoas.Merge(hateoas.AnonymousLinks(), hateoas.RootLinks()))
}

func randomSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("falha ao gerar segredo: %v", err)
	}
	return []byte(hex.EncodeToString(b))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
