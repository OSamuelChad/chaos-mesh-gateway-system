package breaker

import "time"

// WindowView expõe os contadores da janela de amostragem atual do gobreaker.
type WindowView struct {
	Requests             uint32  `json:"requests"`
	TotalSuccesses       uint32  `json:"total_successes"`
	TotalFailures        uint32  `json:"total_failures"`
	ConsecutiveSuccesses uint32  `json:"consecutive_successes"`
	ConsecutiveFailures  uint32  `json:"consecutive_failures"`
	FailureRatio         float64 `json:"failure_ratio"`
}

// MetricsView expõe as métricas acumuladas desde o início do serviço.
type MetricsView struct {
	TotalCalls           uint64  `json:"total_calls"`
	Successes            uint64  `json:"successes"`
	Failures             uint64  `json:"failures"`
	RejectedByBreaker    uint64  `json:"rejected_by_breaker"`
	StateChanges         uint64  `json:"state_changes"`
	AvgUpstreamLatencyMs float64 `json:"avg_upstream_latency_ms"`
}

// ConfigView expõe a configuração do breaker.
type ConfigView struct {
	FailureRatioThreshold float64 `json:"failure_ratio_threshold"`
	MinRequests           uint32  `json:"min_requests"`
	IntervalSeconds       float64 `json:"interval_seconds"`
	OpenTimeoutSeconds    float64 `json:"open_timeout_seconds"`
	HalfOpenMaxRequests   uint32  `json:"half_open_max_requests"`
	RequestTimeoutSeconds float64 `json:"request_timeout_seconds"`
}

// Status é a resposta de GET /resilience/status.
type Status struct {
	Service        string      `json:"service"`
	State          string      `json:"state"`
	Window         WindowView  `json:"window"`
	Metrics        MetricsView `json:"metrics"`
	Config         ConfigView  `json:"config"`
	LastTransition *Transition `json:"last_transition,omitempty"`
	Timestamp      time.Time   `json:"timestamp"`
}

// Status monta a fotografia atual do breaker e das métricas.
func (e *Engine) Status() Status {
	cb := e.cb.Load()
	counts := cb.Counts()

	ratio := 0.0
	if counts.Requests > 0 {
		ratio = float64(counts.TotalFailures) / float64(counts.Requests)
	}

	avg := 0.0
	if hits := e.upstreamHits.Load(); hits > 0 {
		avg = float64(e.latencyMicro.Load()) / float64(hits) / 1000
	}

	return Status{
		Service: "resilience",
		State:   stateName(cb.State()),
		Window: WindowView{
			Requests:             counts.Requests,
			TotalSuccesses:       counts.TotalSuccesses,
			TotalFailures:        counts.TotalFailures,
			ConsecutiveSuccesses: counts.ConsecutiveSuccesses,
			ConsecutiveFailures:  counts.ConsecutiveFailures,
			FailureRatio:         ratio,
		},
		Metrics: MetricsView{
			TotalCalls:           e.calls.Load(),
			Successes:            e.successes.Load(),
			Failures:             e.failures.Load(),
			RejectedByBreaker:    e.rejected.Load(),
			StateChanges:         e.stateChanges.Load(),
			AvgUpstreamLatencyMs: avg,
		},
		Config: ConfigView{
			FailureRatioThreshold: e.cfg.FailureRatio,
			MinRequests:           e.cfg.MinRequests,
			IntervalSeconds:       e.cfg.Interval.Seconds(),
			OpenTimeoutSeconds:    e.cfg.OpenTimeout.Seconds(),
			HalfOpenMaxRequests:   e.cfg.HalfOpenMaxRequests,
			RequestTimeoutSeconds: e.cfg.RequestTimeout.Seconds(),
		},
		LastTransition: e.lastTransition.Load(),
		Timestamp:      time.Now().UTC(),
	}
}
