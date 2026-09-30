package hateoas

import "testing"

func healthyState() MeshState {
	return MeshState{
		PaymentReachable:    true,
		PaymentHealth:       Healthy,
		ResilienceReachable: true,
		BreakerState:        BreakerClosed,
	}
}

func assertRels(t *testing.T, links Links, present []string, absent []string) {
	t.Helper()
	for _, rel := range present {
		if _, ok := links[rel]; !ok {
			t.Errorf("esperava link %q", rel)
		}
	}
	for _, rel := range absent {
		if _, ok := links[rel]; ok {
			t.Errorf("não esperava link %q", rel)
		}
	}
}

func TestPaymentLinksWhenHealthy(t *testing.T) {
	links := ForPayment(healthyState())
	assertRels(t, links,
		[]string{"self", "simulate-traffic", "process-payment", "inject-latency", "inject-faults"},
		[]string{"heal"})
	if links["inject-latency"].Method != "PUT" {
		t.Errorf("inject-latency deveria ser PUT")
	}
}

func TestPaymentLinksWithChaosAndOpenBreaker(t *testing.T) {
	s := healthyState()
	s.ChaosActive, s.LatencyMs, s.FaultRate = true, 2000, 80
	s.BreakerState = BreakerOpen
	assertRels(t, ForPayment(s),
		[]string{"self", "heal", "simulate-traffic"},
		[]string{"inject-latency", "inject-faults", "process-payment"})
}

func TestPaymentLinksWhenUnreachable(t *testing.T) {
	s := healthyState()
	s.PaymentReachable = false
	assertRels(t, ForPayment(s), []string{"self"}, []string{"simulate-traffic", "inject-faults", "heal"})
}

func TestResilienceLinksFollowBreakerState(t *testing.T) {
	assertRels(t, ForResilience(healthyState()), []string{"simulate-traffic"}, []string{"reset-breaker", "heal-payment"})

	s := healthyState()
	s.BreakerState, s.ChaosActive = BreakerOpen, true
	assertRels(t, ForResilience(s), []string{"reset-breaker", "heal-payment"}, nil)

	s.BreakerState = BreakerHalfOpen
	assertRels(t, ForResilience(s), []string{"reset-breaker"}, []string{"heal-payment"})
}

func TestNodeHealth(t *testing.T) {
	s := healthyState()
	if s.GatewayNodeHealth() != Healthy || s.PaymentNodeHealth() != Healthy || s.ResilienceNodeHealth() != Healthy {
		t.Fatal("malha saudável deveria ser HEALTHY")
	}

	s.BreakerState = BreakerHalfOpen
	if s.ResilienceNodeHealth() != Degraded {
		t.Error("HALF-OPEN deveria ser DEGRADED")
	}

	s.BreakerState = BreakerOpen
	if s.ResilienceNodeHealth() != Critical || s.PaymentNodeHealth() != Critical || s.GatewayNodeHealth() != Degraded {
		t.Error("breaker OPEN deveria deixar resilience/payment CRITICAL e gateway DEGRADED")
	}

	s = healthyState()
	s.PaymentHealth = Degraded
	if s.PaymentNodeHealth() != Degraded {
		t.Error("latência alta deveria deixar payment DEGRADED")
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	a := Links{"self": get("/a", "a")}
	b := Links{"self": get("/b", "b")}
	merged := Merge(a, b)
	if merged["self"].Href != "/b" || a["self"].Href != "/a" {
		t.Fatal("Merge deve priorizar o último conjunto sem alterar as entradas")
	}
}
