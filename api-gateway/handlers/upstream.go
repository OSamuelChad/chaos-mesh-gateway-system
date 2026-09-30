package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"api-gateway/hateoas"
)

const (
	statusTimeout   = 2 * time.Second
	maxUpstreamBody = 1 << 20
)

// Upstream encaminha chamadas aos microsserviços pela rede interna do Docker.
type Upstream struct {
	paymentURL    string
	resilienceURL string
	client        *http.Client
}

// NewUpstream cria o cliente dos serviços internos.
func NewUpstream(paymentURL, resilienceURL string) *Upstream {
	return &Upstream{
		paymentURL:    strings.TrimRight(paymentURL, "/"),
		resilienceURL: strings.TrimRight(resilienceURL, "/"),
		client:        &http.Client{Timeout: 15 * time.Second},
	}
}

type upstreamReply struct {
	status     int
	body       []byte
	retryAfter string
}

// UpstreamError indica que o serviço respondeu com status de erro.
type UpstreamError struct {
	Status int
	Body   string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("serviço respondeu %d: %s", e.Status, e.Body)
}

func (u *Upstream) call(ctx context.Context, method, url string, payload []byte) (upstreamReply, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return upstreamReply{}, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return upstreamReply{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamBody))
	if err != nil {
		return upstreamReply{}, err
	}
	return upstreamReply{status: resp.StatusCode, body: data, retryAfter: resp.Header.Get("Retry-After")}, nil
}

// callJSON executa a chamada e decodifica a resposta; status >= 400 vira UpstreamError.
func (u *Upstream) callJSON(ctx context.Context, method, url string, in, out interface{}) error {
	var payload []byte
	if in != nil {
		var err error
		if payload, err = json.Marshal(in); err != nil {
			return err
		}
	}
	reply, err := u.call(ctx, method, url, payload)
	if err != nil {
		return err
	}
	if reply.status >= http.StatusBadRequest {
		return &UpstreamError{Status: reply.status, Body: string(reply.body)}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(reply.body, out)
}

// meshSnapshot agrega o estado atual dos dois serviços.
type meshSnapshot struct {
	payment       *PaymentStatus
	paymentErr    error
	resilience    *ResilienceStatus
	resilienceErr error
}

// snapshot consulta os dois serviços em paralelo.
func (u *Upstream) snapshot(ctx context.Context) meshSnapshot {
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()

	var (
		wg   sync.WaitGroup
		snap meshSnapshot
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		var st PaymentStatus
		if err := u.callJSON(ctx, http.MethodGet, u.paymentURL+"/chaos/status", nil, &st); err != nil {
			snap.paymentErr = err
			return
		}
		snap.payment = &st
	}()
	go func() {
		defer wg.Done()
		var st ResilienceStatus
		if err := u.callJSON(ctx, http.MethodGet, u.resilienceURL+"/resilience/status", nil, &st); err != nil {
			snap.resilienceErr = err
			return
		}
		snap.resilience = &st
	}()
	wg.Wait()
	return snap
}

// state converte o snapshot na entrada das regras HATEOAS.
func (s meshSnapshot) state() hateoas.MeshState {
	st := hateoas.MeshState{BreakerState: hateoas.BreakerUnknown}
	if s.payment != nil {
		st.PaymentReachable = true
		st.PaymentHealth = hateoas.Health(s.payment.Health)
		st.ChaosActive = s.payment.ChaosActive
		st.LatencyMs = s.payment.LatencyMs
		st.FaultRate = s.payment.FaultRate
	}
	if s.resilience != nil {
		st.ResilienceReachable = true
		st.BreakerState = s.resilience.State
	}
	return st
}
