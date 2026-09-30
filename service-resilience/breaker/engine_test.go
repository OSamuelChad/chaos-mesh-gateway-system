package breaker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker"
)

func newTestEngine(t *testing.T, handler http.HandlerFunc) *Engine {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig(srv.URL)
	cfg.OpenTimeout = 200 * time.Millisecond
	return NewEngine(cfg, srv.Client())
}

func TestShouldTrip(t *testing.T) {
	cases := []struct {
		name     string
		counts   gobreaker.Counts
		wantTrip bool
	}{
		{"amostra insuficiente", gobreaker.Counts{Requests: 4, TotalFailures: 4}, false},
		{"exatamente 50% não abre", gobreaker.Counts{Requests: 10, TotalFailures: 5}, false},
		{"acima de 50% abre", gobreaker.Counts{Requests: 10, TotalFailures: 6}, true},
	}
	for _, tc := range cases {
		if got := ShouldTrip(tc.counts, 5, 0.5); got != tc.wantTrip {
			t.Errorf("%s: ShouldTrip = %v, want %v", tc.name, got, tc.wantTrip)
		}
	}
}

func TestProcessPassesThroughSuccess(t *testing.T) {
	e := newTestEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"APPROVED"}`))
	})
	out := e.Process(context.Background(), []byte(`{}`))
	if out.StatusCode != http.StatusCreated || out.Fallback {
		t.Fatalf("outcome inesperado: %+v", out)
	}
	if e.State() != StateClosed {
		t.Fatalf("state = %s, want CLOSED", e.State())
	}
}

func TestBreakerOpensAndReturnsFallback(t *testing.T) {
	var hits atomic.Int32
	e := newTestEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	for i := 0; i < 5; i++ {
		if out := e.Process(context.Background(), []byte(`{}`)); out.StatusCode != http.StatusBadGateway {
			t.Fatalf("chamada %d: status = %d, want 502", i, out.StatusCode)
		}
	}
	if e.State() != StateOpen {
		t.Fatalf("state = %s, want OPEN", e.State())
	}

	out := e.Process(context.Background(), []byte(`{}`))
	if out.StatusCode != http.StatusServiceUnavailable || !out.Fallback {
		t.Fatalf("esperava fallback 503, veio %+v", out)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(out.Body, &body); err != nil || body["fallback"] != true {
		t.Fatalf("payload de fallback inválido: %s", out.Body)
	}
	if hits.Load() != 5 {
		t.Fatalf("breaker aberto não deveria chamar o upstream: hits = %d", hits.Load())
	}
	if got := e.Status().Metrics.RejectedByBreaker; got != 1 {
		t.Fatalf("rejected = %d, want 1", got)
	}
}

func TestBreakerMovesToHalfOpenAfterTimeout(t *testing.T) {
	e := newTestEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	for i := 0; i < 5; i++ {
		e.Process(context.Background(), []byte(`{}`))
	}
	time.Sleep(250 * time.Millisecond)
	if e.State() != StateHalfOpen {
		t.Fatalf("state = %s, want HALF-OPEN", e.State())
	}
}

func TestClientErrorsDoNotTrip(t *testing.T) {
	e := newTestEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	for i := 0; i < 10; i++ {
		e.Process(context.Background(), []byte(`{}`))
	}
	if e.State() != StateClosed {
		t.Fatalf("4xx não deve abrir o breaker, state = %s", e.State())
	}
}

func TestResetClosesBreaker(t *testing.T) {
	e := newTestEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	for i := 0; i < 5; i++ {
		e.Process(context.Background(), []byte(`{}`))
	}
	e.Reset()
	st := e.Status()
	if st.State != StateClosed || st.LastTransition == nil || st.LastTransition.To != StateClosed {
		t.Fatalf("reset não fechou o breaker: %+v", st)
	}
}
