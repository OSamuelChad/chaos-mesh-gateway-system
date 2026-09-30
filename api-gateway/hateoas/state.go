package hateoas

// Health é a saúde de um nó da malha.
type Health string

const (
	Healthy  Health = "HEALTHY"
	Degraded Health = "DEGRADED"
	Critical Health = "CRITICAL"
)

// Estados do Circuit Breaker.
const (
	BreakerClosed   = "CLOSED"
	BreakerOpen     = "OPEN"
	BreakerHalfOpen = "HALF-OPEN"
	BreakerUnknown  = "UNKNOWN"
)

// MeshState é a fotografia da malha usada pelas regras de links e de saúde.
type MeshState struct {
	PaymentReachable bool
	PaymentHealth    Health
	ChaosActive      bool
	LatencyMs        int64
	FaultRate        int64

	ResilienceReachable bool
	BreakerState        string
}

// PaymentNodeHealth: vermelho se inacessível, com breaker aberto ou taxa de erros crítica.
func (s MeshState) PaymentNodeHealth() Health {
	if !s.PaymentReachable || s.BreakerState == BreakerOpen {
		return Critical
	}
	if s.PaymentHealth == "" {
		return Healthy
	}
	return s.PaymentHealth
}

// ResilienceNodeHealth reflete o estado do Circuit Breaker.
func (s MeshState) ResilienceNodeHealth() Health {
	switch {
	case !s.ResilienceReachable, s.BreakerState == BreakerOpen:
		return Critical
	case s.BreakerState == BreakerHalfOpen:
		return Degraded
	default:
		return Healthy
	}
}

// GatewayNodeHealth fica DEGRADED quando algum serviço a jusante está crítico.
func (s MeshState) GatewayNodeHealth() Health {
	if !s.PaymentReachable && !s.ResilienceReachable {
		return Critical
	}
	if s.PaymentNodeHealth() == Critical || s.ResilienceNodeHealth() == Critical {
		return Degraded
	}
	return Healthy
}
