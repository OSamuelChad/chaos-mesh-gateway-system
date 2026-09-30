package main

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const maxAmount = 1_000_000

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type processRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Customer string  `json:"customer"`
}

type processResponse struct {
	TransactionID string    `json:"transaction_id"`
	Status        string    `json:"status"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Customer      string    `json:"customer"`
	ProcessedAt   time.Time `json:"processed_at"`
}

// processHandler simula o processamento de uma cobrança.
func processHandler(c *fiber.Ctx) error {
	var req processRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo JSON inválido")
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		req.Currency = "BRL"
	}
	req.Customer = strings.TrimSpace(req.Customer)

	switch {
	case req.Amount <= 0 || req.Amount > maxAmount:
		return fiber.NewError(fiber.StatusBadRequest, "amount deve ser maior que 0 e no máximo 1.000.000")
	case !currencyPattern.MatchString(req.Currency):
		return fiber.NewError(fiber.StatusBadRequest, "currency deve ser um código ISO de 3 letras")
	case req.Customer == "" || len(req.Customer) > 64:
		return fiber.NewError(fiber.StatusBadRequest, "customer é obrigatório (até 64 caracteres)")
	}

	id, err := newTransactionID()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "falha ao gerar id da transação")
	}

	return c.Status(fiber.StatusCreated).JSON(processResponse{
		TransactionID: id,
		Status:        "APPROVED",
		Amount:        req.Amount,
		Currency:      req.Currency,
		Customer:      req.Customer,
		ProcessedAt:   time.Now().UTC(),
	})
}

func newTransactionID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "txn_" + hex.EncodeToString(b), nil
}
