package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
	"api-gateway/middleware"
)

// RootResponse é o índice navegável da API.
type RootResponse struct {
	Service string `json:"service" example:"api-gateway"`
	Version string `json:"version" example:"1.0.0"`
	hateoas.Resource
}

// Root godoc
//
//	@Summary	Índice da API
//	@Tags		mesh
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	RootResponse
//	@Failure	401	{object}	hateoas.ErrorResponse
//	@Router		/api/v1 [get]
func (h *Handler) Root(c *fiber.Ctx) error {
	return c.JSON(RootResponse{Service: "api-gateway", Version: Version, Resource: hateoas.Resource{Links: hateoas.RootLinks()}})
}

// Topology godoc
//
//	@Summary		Topologia da malha
//	@Description	Agrega o estado do Payment e do Resilience Service e devolve nós, arestas e links.
//	@Tags			mesh
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	TopologyResponse
//	@Failure		401	{object}	hateoas.ErrorResponse
//	@Router			/api/v1/mesh/topology [get]
func (h *Handler) Topology(c *fiber.Ctx) error {
	snap := h.up.snapshot(c.UserContext())
	st := snap.state()

	nodes := []TopologyNode{
		{
			ID: "gateway", Label: "API Gateway", Role: "Entrada · JWT · HATEOAS",
			Health: st.GatewayNodeHealth(), Reachable: true,
			Summary:  h.gatewaySummary(),
			Resource: hateoas.Resource{Links: hateoas.Links{"self": {Href: hateoas.PathGateway, Method: "GET", Title: "Detalhar"}}},
		},
		{
			ID: "resilience", Label: "Resilience Service", Role: "Circuit Breaker",
			Health: st.ResilienceNodeHealth(), Reachable: st.ResilienceReachable,
			Summary:  resilienceSummary(snap.resilience),
			Resource: hateoas.Resource{Links: hateoas.Links{"self": {Href: hateoas.PathResilience, Method: "GET", Title: "Detalhar"}}},
		},
		{
			ID: "payment", Label: "Payment Service", Role: "Processamento · Caos",
			Health: st.PaymentNodeHealth(), Reachable: st.PaymentReachable,
			Summary:  paymentSummary(snap.payment),
			Resource: hateoas.Resource{Links: hateoas.Links{"self": {Href: hateoas.PathPayment, Method: "GET", Title: "Detalhar"}}},
		},
	}

	edges := []TopologyEdge{
		{
			ID: "gateway-resilience", Source: "gateway", Target: "resilience", Label: "HTTP /resilience",
			Animated: st.ResilienceReachable, Health: st.ResilienceNodeHealth(),
		},
		{
			ID: "resilience-payment", Source: "resilience", Target: "payment", Label: "breaker: " + st.BreakerState,
			Animated: st.PaymentReachable && st.BreakerState != hateoas.BreakerOpen, Health: st.PaymentNodeHealth(),
		},
	}

	return c.JSON(TopologyResponse{
		Nodes: nodes, Edges: edges, BreakerState: st.BreakerState, Timestamp: time.Now().UTC(),
		Resource: hateoas.Resource{Links: hateoas.ForTopology(st)},
	})
}

// GatewayNode godoc
//
//	@Summary	Detalhe do API Gateway
//	@Tags		services
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	NodeResponse
//	@Failure	401	{object}	hateoas.ErrorResponse
//	@Router		/api/v1/services/gateway [get]
func (h *Handler) GatewayNode(c *fiber.Ctx) error {
	st := h.up.snapshot(c.UserContext()).state()
	user, _ := c.Locals(middleware.LocalUser).(string)
	return c.JSON(NodeResponse{
		ID: "gateway", Name: "API Gateway", Health: st.GatewayNodeHealth(), Reachable: true,
		Summary:  h.gatewaySummary(),
		Gateway:  &GatewayInfo{Version: Version, UptimeSeconds: time.Since(h.startedAt).Seconds(), AuthenticatedAs: user},
		Resource: hateoas.Resource{Links: hateoas.ForGateway(st)},
	})
}

// PaymentNode godoc
//
//	@Summary		Detalhe do Payment Service
//	@Description	Os links retornados variam com o caos ativo e com o estado do Circuit Breaker.
//	@Tags			services
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	NodeResponse
//	@Failure		401	{object}	hateoas.ErrorResponse
//	@Router			/api/v1/services/payment [get]
func (h *Handler) PaymentNode(c *fiber.Ctx) error {
	return c.JSON(h.paymentNode(c))
}

func (h *Handler) paymentNode(c *fiber.Ctx) NodeResponse {
	snap := h.up.snapshot(c.UserContext())
	st := snap.state()
	resp := NodeResponse{
		ID: "payment", Name: "Payment Service", Health: st.PaymentNodeHealth(), Reachable: st.PaymentReachable,
		Summary: paymentSummary(snap.payment), Payment: snap.payment,
		Resource: hateoas.Resource{Links: hateoas.ForPayment(st)},
	}
	if snap.paymentErr != nil {
		resp.Error = "payment service inacessível"
	}
	return resp
}

// ResilienceNode godoc
//
//	@Summary		Detalhe do Resilience Service
//	@Description	Estado do Circuit Breaker (CLOSED, OPEN, HALF-OPEN) e métricas.
//	@Tags			services
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	NodeResponse
//	@Failure		401	{object}	hateoas.ErrorResponse
//	@Router			/api/v1/services/resilience [get]
func (h *Handler) ResilienceNode(c *fiber.Ctx) error {
	return c.JSON(h.resilienceNode(c))
}

func (h *Handler) resilienceNode(c *fiber.Ctx) NodeResponse {
	snap := h.up.snapshot(c.UserContext())
	st := snap.state()
	resp := NodeResponse{
		ID: "resilience", Name: "Resilience Service", Health: st.ResilienceNodeHealth(), Reachable: st.ResilienceReachable,
		Summary: resilienceSummary(snap.resilience), Resilience: snap.resilience,
		Resource: hateoas.Resource{Links: hateoas.ForResilience(st)},
	}
	if snap.resilienceErr != nil {
		resp.Error = "resilience service inacessível"
	}
	return resp
}

func (h *Handler) gatewaySummary() []Metric {
	return []Metric{
		{Label: "Versão", Value: Version},
		{Label: "Uptime", Value: time.Since(h.startedAt).Round(time.Second).String()},
	}
}

func paymentSummary(p *PaymentStatus) []Metric {
	if p == nil {
		return []Metric{{Label: "Status", Value: "inacessível"}}
	}
	return []Metric{
		{Label: "Latência", Value: fmt.Sprintf("%d ms", p.LatencyMs)},
		{Label: "Erros", Value: fmt.Sprintf("%d%%", p.FaultRate)},
		{Label: "Requisições", Value: fmt.Sprintf("%d", p.TotalRequests)},
	}
}

func resilienceSummary(r *ResilienceStatus) []Metric {
	if r == nil {
		return []Metric{{Label: "Status", Value: "inacessível"}}
	}
	return []Metric{
		{Label: "Breaker", Value: r.State},
		{Label: "Falhas", Value: fmt.Sprintf("%d", r.Metrics.Failures)},
		{Label: "Fallbacks", Value: fmt.Sprintf("%d", r.Metrics.RejectedByBreaker)},
	}
}
