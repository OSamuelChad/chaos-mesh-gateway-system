// Package chaos implementa um controlador de caos programável que injeta
// latência artificial e falhas HTTP 500 nas requisições de negócio.
//
// Todo o estado é mantido com operações atômicas (sync/atomic), então as
// rotas administrativas podem alterar a configuração enquanto o middleware
// processa requisições concorrentes, sem race conditions.
package chaos

import (
	"errors"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

const (
	// MaxLatencyMs é a maior latência artificial aceita (10 segundos).
	MaxLatencyMs = 10_000
	// MaxFaultRate é a maior taxa de falhas aceita (100%).
	MaxFaultRate = 100
	// DegradedLatencyMs é a latência a partir da qual o serviço é considerado DEGRADED.
	DegradedLatencyMs = 1_000
	// CriticalFaultRate é a taxa de falhas a partir da qual o serviço é considerado CRITICAL.
	CriticalFaultRate = 50
)

// Health representa o estado de saúde reportado pelo serviço.
type Health string

const (
	Healthy  Health = "HEALTHY"
	Degraded Health = "DEGRADED"
	Critical Health = "CRITICAL"
)

var (
	ErrInvalidLatency   = errors.New("latency_ms deve estar entre 0 e 10000")
	ErrInvalidFaultRate = errors.New("rate deve estar entre 0 e 100")
)

// Controller guarda a configuração de caos e as métricas de execução.
type Controller struct {
	latencyMs atomic.Int64
	faultRate atomic.Int64

	requests        atomic.Uint64
	injectedFaults  atomic.Uint64
	delayedRequests atomic.Uint64

	roll  func() int64
	sleep func(time.Duration)
}

// NewController cria um controlador em operação normal (sem caos).
func NewController() *Controller {
	return &Controller{
		roll:  func() int64 { return rand.Int64N(MaxFaultRate) },
		sleep: time.Sleep,
	}
}

// SetLatency define a latência artificial em milissegundos.
func (c *Controller) SetLatency(ms int64) error {
	if ms < 0 || ms > MaxLatencyMs {
		return ErrInvalidLatency
	}
	c.latencyMs.Store(ms)
	return nil
}

// SetFaultRate define a porcentagem (0-100) de requisições que falharão com HTTP 500.
func (c *Controller) SetFaultRate(rate int64) error {
	if rate < 0 || rate > MaxFaultRate {
		return ErrInvalidFaultRate
	}
	c.faultRate.Store(rate)
	return nil
}

// Reset restaura a operação normal. As métricas acumuladas são preservadas.
func (c *Controller) Reset() {
	c.latencyMs.Store(0)
	c.faultRate.Store(0)
}

// Snapshot é a visão pública do estado do controlador.
type Snapshot struct {
	Health          Health `json:"health"`
	ChaosActive     bool   `json:"chaos_active"`
	LatencyMs       int64  `json:"latency_ms"`
	FaultRate       int64  `json:"fault_rate"`
	TotalRequests   uint64 `json:"total_requests"`
	InjectedFaults  uint64 `json:"injected_faults"`
	DelayedRequests uint64 `json:"delayed_requests"`
}

// Snapshot retorna uma cópia consistente o suficiente para monitoramento.
func (c *Controller) Snapshot() Snapshot {
	latency := c.latencyMs.Load()
	rate := c.faultRate.Load()
	return Snapshot{
		Health:          HealthFor(latency, rate),
		ChaosActive:     latency > 0 || rate > 0,
		LatencyMs:       latency,
		FaultRate:       rate,
		TotalRequests:   c.requests.Load(),
		InjectedFaults:  c.injectedFaults.Load(),
		DelayedRequests: c.delayedRequests.Load(),
	}
}

// HealthFor calcula a saúde a partir da configuração de caos.
func HealthFor(latencyMs, faultRate int64) Health {
	switch {
	case faultRate >= CriticalFaultRate:
		return Critical
	case faultRate > 0 || latencyMs >= DegradedLatencyMs:
		return Degraded
	default:
		return Healthy
	}
}

// apply aplica o caos configurado e informa se a requisição deve falhar.
func (c *Controller) apply() (fail bool) {
	c.requests.Add(1)

	if latency := c.latencyMs.Load(); latency > 0 {
		c.delayedRequests.Add(1)
		c.sleep(time.Duration(latency) * time.Millisecond)
	}

	if rate := c.faultRate.Load(); rate > 0 && c.roll() < rate {
		c.injectedFaults.Add(1)
		return true
	}
	return false
}
