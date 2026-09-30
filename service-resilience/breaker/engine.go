// Package breaker envolve as chamadas ao service-payment com um Circuit Breaker
// (sony/gobreaker) e fornece fallback graceful quando o circuito está aberto.
package breaker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker"
)

// Estados expostos pela API.
const (
	StateClosed   = "CLOSED"
	StateOpen     = "OPEN"
	StateHalfOpen = "HALF-OPEN"
)

const maxUpstreamBody = 1 << 20 // 1 MiB

// errUpstream marca respostas 5xx do service-payment como falha para o breaker.
var errUpstream = errors.New("service-payment respondeu com erro 5xx")

// Config parametriza o Circuit Breaker.
type Config struct {
	PaymentURL          string
	RequestTimeout      time.Duration
	FailureRatio        float64       // abre quando a taxa de falhas EXCEDER este valor
	MinRequests         uint32        // amostra mínima da janela antes de avaliar a taxa
	Interval            time.Duration // janela de amostragem (contadores zeram em CLOSED)
	OpenTimeout         time.Duration // tempo em OPEN antes de ir para HALF-OPEN
	HalfOpenMaxRequests uint32        // requisições de teste permitidas em HALF-OPEN
}

// DefaultConfig retorna a configuração padrão do trabalho: abre com > 50% de falhas.
func DefaultConfig(paymentURL string) Config {
	return Config{
		PaymentURL:          strings.TrimRight(paymentURL, "/"),
		RequestTimeout:      3 * time.Second,
		FailureRatio:        0.5,
		MinRequests:         5,
		Interval:            30 * time.Second,
		OpenTimeout:         10 * time.Second,
		HalfOpenMaxRequests: 3,
	}
}

// Transition registra uma mudança de estado do breaker.
type Transition struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	At   time.Time `json:"at"`
}

// Outcome é o resultado de uma requisição processada pelo engine.
type Outcome struct {
	StatusCode int
	Body       json.RawMessage
	Fallback   bool
	RetryAfter int
}

type upstreamResponse struct {
	status int
	body   []byte
}

// Engine executa chamadas ao service-payment protegidas pelo Circuit Breaker.
type Engine struct {
	cfg    Config
	client *http.Client
	cb     atomic.Pointer[gobreaker.CircuitBreaker]

	calls        atomic.Uint64
	successes    atomic.Uint64
	failures     atomic.Uint64
	rejected     atomic.Uint64
	stateChanges atomic.Uint64
	latencyMicro atomic.Uint64 // soma das latências das chamadas que chegaram ao upstream
	upstreamHits atomic.Uint64

	lastTransition atomic.Pointer[Transition]
}

// NewEngine cria o engine com um breaker em estado CLOSED.
func NewEngine(cfg Config, client *http.Client) *Engine {
	if client == nil {
		client = &http.Client{Timeout: cfg.RequestTimeout}
	}
	e := &Engine{cfg: cfg, client: client}
	e.cb.Store(e.newBreaker())
	return e
}

func (e *Engine) newBreaker() *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "service-payment",
		MaxRequests: e.cfg.HalfOpenMaxRequests,
		Interval:    e.cfg.Interval,
		Timeout:     e.cfg.OpenTimeout,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return ShouldTrip(c, e.cfg.MinRequests, e.cfg.FailureRatio)
		},
		OnStateChange: func(_ string, from, to gobreaker.State) {
			e.recordTransition(stateName(from), stateName(to))
		},
	})
}

// ShouldTrip decide se o breaker deve abrir: amostra mínima atingida e taxa de falhas > limite.
func ShouldTrip(c gobreaker.Counts, minRequests uint32, ratio float64) bool {
	if c.Requests < minRequests {
		return false
	}
	return float64(c.TotalFailures)/float64(c.Requests) > ratio
}

func (e *Engine) recordTransition(from, to string) {
	e.stateChanges.Add(1)
	e.lastTransition.Store(&Transition{From: from, To: to, At: time.Now().UTC()})
	log.Printf("circuit breaker: %s -> %s", from, to)
}

// Process envia a cobrança ao service-payment através do breaker.
func (e *Engine) Process(ctx context.Context, payload []byte) Outcome {
	e.calls.Add(1)

	result, err := e.cb.Load().Execute(func() (interface{}, error) {
		return e.callPayment(ctx, payload)
	})

	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		e.rejected.Add(1)
		return e.fallback()
	}

	if err != nil {
		e.failures.Add(1)
		return upstreamFailure(result, err)
	}

	e.successes.Add(1)
	res := result.(*upstreamResponse)
	return Outcome{StatusCode: res.status, Body: res.body}
}

func (e *Engine) callPayment(ctx context.Context, payload []byte) (*upstreamResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.PaymentURL+"/process", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := e.client.Do(req)
	e.upstreamHits.Add(1)
	e.latencyMicro.Add(uint64(time.Since(start).Microseconds()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamBody))
	if err != nil {
		return nil, err
	}
	res := &upstreamResponse{status: resp.StatusCode, body: body}
	if resp.StatusCode >= http.StatusInternalServerError {
		return res, errUpstream
	}
	// Erros 4xx são de validação do cliente: não contam como falha do serviço.
	return res, nil
}

func upstreamFailure(result interface{}, err error) Outcome {
	var timeout interface{ Timeout() bool }
	status := http.StatusBadGateway
	if errors.As(err, &timeout) && timeout.Timeout() {
		status = http.StatusGatewayTimeout
	}

	payload := map[string]interface{}{
		"error":    "falha ao processar pagamento no service-payment",
		"detail":   err.Error(),
		"fallback": false,
	}
	if res, ok := result.(*upstreamResponse); ok && res != nil {
		payload["upstream_status"] = res.status
		var upstreamBody interface{}
		if json.Unmarshal(res.body, &upstreamBody) == nil {
			payload["upstream_body"] = upstreamBody
		}
	}
	body, _ := json.Marshal(payload)
	return Outcome{StatusCode: status, Body: body}
}

func (e *Engine) fallback() Outcome {
	retryAfter := e.retryAfterSeconds()
	body, _ := json.Marshal(map[string]interface{}{
		"fallback":            true,
		"status":              "UNAVAILABLE",
		"breaker_state":       e.State(),
		"message":             "O serviço de pagamentos está temporariamente indisponível. Sua cobrança NÃO foi processada; tente novamente em instantes.",
		"retry_after_seconds": retryAfter,
	})
	return Outcome{StatusCode: http.StatusServiceUnavailable, Body: body, Fallback: true, RetryAfter: retryAfter}
}

func (e *Engine) retryAfterSeconds() int {
	t := e.lastTransition.Load()
	if t == nil || t.To != StateOpen {
		return 1
	}
	remaining := time.Until(t.At.Add(e.cfg.OpenTimeout)).Seconds()
	return int(math.Max(1, math.Ceil(remaining)))
}

// State retorna o estado atual: CLOSED, OPEN ou HALF-OPEN.
func (e *Engine) State() string {
	return stateName(e.cb.Load().State())
}

// Reset força o breaker de volta para CLOSED, zerando a janela de amostragem.
func (e *Engine) Reset() {
	previous := e.State()
	e.cb.Store(e.newBreaker())
	if previous != StateClosed {
		e.recordTransition(previous, StateClosed)
	}
}

func stateName(s gobreaker.State) string {
	switch s {
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	case gobreaker.StateClosed:
		return StateClosed
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}
