package hateoas

// Rotas públicas do gateway usadas na construção dos links.
const (
	APIBase = "/api/v1"

	PathRoot     = APIBase
	PathLogin    = APIBase + "/auth/login"
	PathTopology = APIBase + "/mesh/topology"
	PathPayments = APIBase + "/payments"
	PathTraffic  = APIBase + "/traffic/simulate"

	PathGateway = APIBase + "/services/gateway"

	PathPayment        = APIBase + "/services/payment"
	PathPaymentLatency = PathPayment + "/chaos/latency"
	PathPaymentFaults  = PathPayment + "/chaos/faults"
	PathPaymentHeal    = PathPayment + "/chaos/reset"

	PathResilience      = APIBase + "/services/resilience"
	PathResilienceReset = PathResilience + "/breaker/reset"

	PathSwagger = "/swagger/index.html"
)
