package chaos

import (
	"sync"
	"testing"
	"time"
)

func newTestController(roll int64) (*Controller, *time.Duration) {
	var slept time.Duration
	c := NewController()
	c.roll = func() int64 { return roll }
	c.sleep = func(d time.Duration) { slept += d }
	return c, &slept
}

func TestSetLatencyValidatesRange(t *testing.T) {
	c := NewController()
	if err := c.SetLatency(-1); err == nil {
		t.Fatal("esperava erro para latência negativa")
	}
	if err := c.SetLatency(MaxLatencyMs + 1); err == nil {
		t.Fatal("esperava erro para latência acima do máximo")
	}
	if err := c.SetLatency(2000); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got := c.Snapshot().LatencyMs; got != 2000 {
		t.Fatalf("latency = %d, want 2000", got)
	}
}

func TestSetFaultRateValidatesRange(t *testing.T) {
	c := NewController()
	if err := c.SetFaultRate(101); err == nil {
		t.Fatal("esperava erro para taxa acima de 100")
	}
	if err := c.SetFaultRate(80); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestApplyInjectsLatencyAndFaults(t *testing.T) {
	c, slept := newTestController(10)
	_ = c.SetLatency(2000)
	_ = c.SetFaultRate(80)

	if fail := c.apply(); !fail {
		t.Fatal("roll 10 < rate 80: esperava falha")
	}
	if *slept != 2*time.Second {
		t.Fatalf("slept = %v, want 2s", *slept)
	}
	snap := c.Snapshot()
	if snap.InjectedFaults != 1 || snap.DelayedRequests != 1 || snap.TotalRequests != 1 {
		t.Fatalf("métricas inesperadas: %+v", snap)
	}
}

func TestApplyPassesWhenRollAboveRate(t *testing.T) {
	c, _ := newTestController(90)
	_ = c.SetFaultRate(80)
	if c.apply() {
		t.Fatal("roll 90 >= rate 80: não esperava falha")
	}
}

func TestResetRestoresNormalOperation(t *testing.T) {
	c := NewController()
	_ = c.SetLatency(500)
	_ = c.SetFaultRate(30)
	c.Reset()
	snap := c.Snapshot()
	if snap.ChaosActive || snap.Health != Healthy {
		t.Fatalf("esperava operação normal após reset: %+v", snap)
	}
}

func TestHealthFor(t *testing.T) {
	cases := []struct {
		latency, rate int64
		want          Health
	}{
		{0, 0, Healthy},
		{500, 0, Healthy},
		{2000, 0, Degraded},
		{0, 20, Degraded},
		{0, 50, Critical},
		{2000, 80, Critical},
	}
	for _, tc := range cases {
		if got := HealthFor(tc.latency, tc.rate); got != tc.want {
			t.Errorf("HealthFor(%d, %d) = %s, want %s", tc.latency, tc.rate, got, tc.want)
		}
	}
}

func TestConcurrentAccessIsRaceFree(t *testing.T) {
	c, _ := newTestController(0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) { defer wg.Done(); _ = c.SetFaultRate(int64(i % 100)) }(i)
		go func() { defer wg.Done(); c.apply() }()
	}
	wg.Wait()
	if got := c.Snapshot().TotalRequests; got != 50 {
		t.Fatalf("total_requests = %d, want 50", got)
	}
}
