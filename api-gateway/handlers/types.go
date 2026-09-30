package handlers

import (
	"time"

	"api-gateway/hateoas"
)

// PaymentStatus espelha GET /chaos/status do service-payment.
type PaymentStatus struct {
	Health          string `json:"health" example:"HEALTHY"`
	ChaosActive     bool   `json:"chaos_active"`
	LatencyMs       int64  `json:"latency_ms" example:"0"`
	FaultRate       int64  `json:"fault_rate" example:"0"`
	TotalRequests   uint64 `json:"total_requests"`
	InjectedFaults  uint64 `json:"injected_faults"`
	DelayedRequests uint64 `json:"delayed_requests"`
}

// BreakerWindow espelha a janela de amostragem do Circuit Breaker.
type BreakerWindow struct {
	Requests             uint32  `json:"requests"`
	TotalSuccesses       uint32  `json:"total_successes"`
	TotalFailures        uint32  `json:"total_failures"`
	ConsecutiveSuccesses uint32  `json:"consecutive_successes"`
	ConsecutiveFailures  uint32  `json:"consecutive_failures"`
	FailureRatio         float64 `json:"failure_ratio"`
}

// BreakerMetrics espelha as métricas acumuladas do service-resilience.
type BreakerMetrics struct {
	TotalCalls           uint64  `json:"total_calls"`
	Successes            uint64  `json:"successes"`
	Failures             uint64  `json:"failures"`
	RejectedByBreaker    uint64  `json:"rejected_by_breaker"`
	StateChanges         uint64  `json:"state_changes"`
	AvgUpstreamLatencyMs float64 `json:"avg_upstream_latency_ms"`
}

// BreakerConfig espelha a configuração do Circuit Breaker.
type BreakerConfig struct {
	FailureRatioThreshold float64 `json:"failure_ratio_threshold"`
	MinRequests           uint32  `json:"min_requests"`
	IntervalSeconds       float64 `json:"interval_seconds"`
	OpenTimeoutSeconds    float64 `json:"open_timeout_seconds"`
	HalfOpenMaxRequests   uint32  `json:"half_open_max_requests"`
	RequestTimeoutSeconds float64 `json:"request_timeout_seconds"`
}

// BreakerTransition espelha a última mudança de estado.
type BreakerTransition struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	At   time.Time `json:"at"`
}

// ResilienceStatus espelha GET /resilience/status do service-resilience.
type ResilienceStatus struct {
	State          string             `json:"state" example:"CLOSED"`
	Window         BreakerWindow      `json:"window"`
	Metrics        BreakerMetrics     `json:"metrics"`
	Config         BreakerConfig      `json:"config"`
	LastTransition *BreakerTransition `json:"last_transition,omitempty"`
}

// GatewayInfo descreve o próprio gateway.
type GatewayInfo struct {
	Version         string  `json:"version" example:"1.0.0"`
	UptimeSeconds   float64 `json:"uptime_seconds"`
	AuthenticatedAs string  `json:"authenticated_as" example:"admin"`
}

// Metric é um par rótulo/valor exibido nos cartões dos nós.
type Metric struct {
	Label string `json:"label" example:"Latência"`
	Value string `json:"value" example:"0 ms"`
}

// NodeResponse é a resposta de detalhe de um nó da malha.
type NodeResponse struct {
	ID         string            `json:"id" example:"payment"`
	Name       string            `json:"name" example:"Payment Service"`
	Health     hateoas.Health    `json:"health" example:"HEALTHY"`
	Reachable  bool              `json:"reachable"`
	Error      string            `json:"error,omitempty"`
	Summary    []Metric          `json:"summary"`
	Gateway    *GatewayInfo      `json:"gateway,omitempty"`
	Payment    *PaymentStatus    `json:"payment,omitempty"`
	Resilience *ResilienceStatus `json:"resilience,omitempty"`
	hateoas.Resource
}

// TopologyNode é um nó do grafo.
type TopologyNode struct {
	ID        string         `json:"id" example:"gateway"`
	Label     string         `json:"label" example:"API Gateway"`
	Role      string         `json:"role" example:"Entrada · JWT · HATEOAS"`
	Health    hateoas.Health `json:"health" example:"HEALTHY"`
	Reachable bool           `json:"reachable"`
	Summary   []Metric       `json:"summary"`
	hateoas.Resource
}

// TopologyEdge é uma aresta do grafo.
type TopologyEdge struct {
	ID       string         `json:"id" example:"gateway-resilience"`
	Source   string         `json:"source" example:"gateway"`
	Target   string         `json:"target" example:"resilience"`
	Label    string         `json:"label" example:"HTTP"`
	Animated bool           `json:"animated"`
	Health   hateoas.Health `json:"health" example:"HEALTHY"`
}

// TopologyResponse é a resposta do agregador da malha.
type TopologyResponse struct {
	Nodes        []TopologyNode `json:"nodes"`
	Edges        []TopologyEdge `json:"edges"`
	BreakerState string         `json:"breaker_state" example:"CLOSED"`
	Timestamp    time.Time      `json:"timestamp"`
	hateoas.Resource
}

// LatencyRequest define a latência artificial do service-payment.
type LatencyRequest struct {
	LatencyMs *int64 `json:"latency_ms" example:"2000"`
}

// FaultsRequest define a taxa de erros do service-payment.
type FaultsRequest struct {
	Rate *int64 `json:"rate" example:"80"`
}

// TrafficRequest parametriza a simulação de tráfego.
type TrafficRequest struct {
	Requests    int `json:"requests" example:"20"`
	Concurrency int `json:"concurrency" example:"4"`
}

// TrafficResponse resume a simulação de tráfego.
type TrafficResponse struct {
	Requested          int            `json:"requested"`
	Succeeded          int            `json:"succeeded"`
	Failed             int            `json:"failed"`
	Fallbacks          int            `json:"fallbacks"`
	StatusCodes        map[string]int `json:"status_codes"`
	AvgLatencyMs       float64        `json:"avg_latency_ms"`
	DurationMs         int64          `json:"duration_ms"`
	BreakerStateBefore string         `json:"breaker_state_before" example:"CLOSED"`
	BreakerStateAfter  string         `json:"breaker_state_after" example:"OPEN"`
	hateoas.Resource
}

// PaymentResult envolve a resposta do fluxo de pagamento.
type PaymentResult struct {
	StatusCode int         `json:"status_code" example:"201"`
	Fallback   bool        `json:"fallback"`
	Result     interface{} `json:"result" swaggertype:"object"`
	hateoas.Resource
}
