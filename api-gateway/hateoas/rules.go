package hateoas

// Valores sugeridos nos payloads das ações de demonstração.
const (
	DemoLatencyMs       = 2000
	DemoFaultRate       = 80
	DemoTrafficRequests = 20
)

// DemoPayment é o payload de exemplo para processar uma cobrança.
var DemoPayment = map[string]interface{}{"amount": 149.90, "currency": "BRL", "customer": "cliente-demo"}

func trafficLink() Link {
	return post(PathTraffic, "Simular Tráfego", map[string]int{"requests": DemoTrafficRequests})
}

// RootLinks são os pontos de entrada da API.
func RootLinks() Links {
	return Links{
		"self":       get(PathRoot, "Índice da API"),
		"topology":   get(PathTopology, "Topologia da malha"),
		"gateway":    get(PathGateway, "API Gateway"),
		"payment":    get(PathPayment, "Payment Service"),
		"resilience": get(PathResilience, "Resilience Service"),
		"docs":       {Href: PathSwagger, Method: "GET", Title: "Documentação Swagger", Type: "text/html"},
	}
}

// AnonymousLinks são oferecidos a clientes não autenticados.
func AnonymousLinks() Links {
	return Links{
		"login": post(PathLogin, "Autenticar", map[string]string{"username": "admin", "password": "password"}),
		"docs":  {Href: PathSwagger, Method: "GET", Title: "Documentação Swagger", Type: "text/html"},
	}
}

// ForGateway monta as ações do nó API Gateway.
func ForGateway(s MeshState) Links {
	links := Links{
		"self":     get(PathGateway, "Atualizar status"),
		"topology": get(PathTopology, "Ver topologia"),
		"docs":     {Href: PathSwagger, Method: "GET", Title: "Documentação Swagger", Type: "text/html"},
	}
	if s.ResilienceReachable {
		links["simulate-traffic"] = trafficLink()
	}
	return links
}

// ForPayment monta as ações do nó Payment Service de acordo com o caos ativo e o breaker.
func ForPayment(s MeshState) Links {
	links := Links{
		"self":     get(PathPayment, "Atualizar status"),
		"topology": get(PathTopology, "Ver topologia"),
	}
	if !s.PaymentReachable {
		return links
	}

	if s.ResilienceReachable {
		links["simulate-traffic"] = trafficLink()
		// Com o circuito aberto a cobrança seria recusada: a ação some da interface.
		if s.BreakerState != BreakerOpen {
			links["process-payment"] = post(PathPayments, "Processar 1 cobrança", DemoPayment)
		}
	}
	if s.LatencyMs < DemoLatencyMs {
		links["inject-latency"] = put(PathPaymentLatency, "Injetar 2s de Latência", map[string]int{"latency_ms": DemoLatencyMs})
	}
	if s.FaultRate < DemoFaultRate {
		links["inject-faults"] = put(PathPaymentFaults, "Injetar 80% de Erros", map[string]int{"rate": DemoFaultRate})
	}
	if s.ChaosActive {
		links["heal"] = post(PathPaymentHeal, "Restaurar / Heal", nil)
	}
	return links
}

// ForResilience monta as ações do nó Resilience Service de acordo com o estado do breaker.
func ForResilience(s MeshState) Links {
	links := Links{
		"self":     get(PathResilience, "Atualizar status"),
		"topology": get(PathTopology, "Ver topologia"),
	}
	if !s.ResilienceReachable {
		return links
	}

	links["simulate-traffic"] = trafficLink()
	if s.BreakerState != BreakerClosed {
		links["reset-breaker"] = post(PathResilienceReset, "Fechar Circuit Breaker (reset)", nil)
	}
	if s.BreakerState == BreakerOpen && s.ChaosActive {
		links["heal-payment"] = post(PathPaymentHeal, "Restaurar Payment (Heal)", nil)
	}
	return links
}

// ForTopology monta os links da visão geral da malha.
func ForTopology(s MeshState) Links {
	links := Links{
		"self":       get(PathTopology, "Atualizar topologia"),
		"gateway":    get(PathGateway, "API Gateway"),
		"payment":    get(PathPayment, "Payment Service"),
		"resilience": get(PathResilience, "Resilience Service"),
	}
	if s.ResilienceReachable {
		links["simulate-traffic"] = trafficLink()
	}
	return links
}
