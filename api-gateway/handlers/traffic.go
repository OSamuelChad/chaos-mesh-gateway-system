package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-gateway/hateoas"
)

const (
	defaultTrafficRequests    = 20
	maxTrafficRequests        = 100
	defaultTrafficConcurrency = 4
	maxTrafficConcurrency     = 10
)

type trafficSample struct {
	status   int
	fallback bool
	latency  time.Duration
}

// SimulateTraffic godoc
//
//	@Summary		Simular tráfego
//	@Description	Dispara N cobranças (1-100) pelo fluxo Gateway → Resilience → Payment e resume o resultado.
//	@Tags			traffic
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		TrafficRequest	false	"Quantidade e concorrência"
//	@Success		200		{object}	TrafficResponse
//	@Failure		400		{object}	hateoas.ErrorResponse
//	@Failure		401		{object}	hateoas.ErrorResponse
//	@Router			/api/v1/traffic/simulate [post]
func (h *Handler) SimulateTraffic(c *fiber.Ctx) error {
	req := TrafficRequest{Requests: defaultTrafficRequests, Concurrency: defaultTrafficConcurrency}
	if len(c.Body()) > 0 {
		if err := json.Unmarshal(c.Body(), &req); err != nil {
			return hateoas.Error(c, fiber.StatusBadRequest, "corpo JSON inválido", hateoas.ForTopology(h.up.snapshot(c.UserContext()).state()))
		}
	}
	if req.Concurrency == 0 {
		req.Concurrency = defaultTrafficConcurrency
	}
	if req.Requests < 1 || req.Requests > maxTrafficRequests || req.Concurrency < 1 || req.Concurrency > maxTrafficConcurrency {
		return hateoas.Error(c, fiber.StatusBadRequest, "requests deve estar entre 1 e 100 e concurrency entre 1 e 10",
			hateoas.ForTopology(h.up.snapshot(c.UserContext()).state()))
	}

	ctx := c.UserContext()
	before := h.up.snapshot(ctx).state().BreakerState
	start := time.Now()
	samples := h.fire(ctx, req.Requests, req.Concurrency)
	duration := time.Since(start)
	after := h.up.snapshot(ctx).state()

	resp := summarize(samples)
	resp.Requested = req.Requests
	resp.DurationMs = duration.Milliseconds()
	resp.BreakerStateBefore = before
	resp.BreakerStateAfter = after.BreakerState
	resp.Links = hateoas.Merge(hateoas.ForTopology(after), hateoas.ForResilience(after))
	return c.JSON(resp)
}

// fire envia as requisições com concorrência limitada; cada goroutine escreve apenas no próprio índice.
func (h *Handler) fire(ctx context.Context, n, concurrency int) []trafficSample {
	samples := make([]trafficSample, n)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			samples[i] = h.sendOne(ctx, i)
		}(i)
	}
	wg.Wait()
	return samples
}

func (h *Handler) sendOne(ctx context.Context, i int) trafficSample {
	payload, _ := json.Marshal(map[string]interface{}{
		"amount":   10 + float64(i),
		"currency": "BRL",
		"customer": fmt.Sprintf("trafego-sim-%03d", i+1),
	})
	start := time.Now()
	reply, err := h.up.call(ctx, http.MethodPost, h.up.resilienceURL+"/resilience/process", payload)
	sample := trafficSample{latency: time.Since(start)}
	if err != nil {
		sample.status = http.StatusBadGateway
		return sample
	}
	sample.status = reply.status
	var body struct {
		Fallback bool `json:"fallback"`
	}
	if json.Unmarshal(reply.body, &body) == nil {
		sample.fallback = body.Fallback
	}
	return sample
}

func summarize(samples []trafficSample) TrafficResponse {
	resp := TrafficResponse{StatusCodes: map[string]int{}}
	var total time.Duration
	for _, s := range samples {
		total += s.latency
		resp.StatusCodes[strconv.Itoa(s.status)]++
		switch {
		case s.status < http.StatusBadRequest:
			resp.Succeeded++
		case s.fallback:
			resp.Fallbacks++
		default:
			resp.Failed++
		}
	}
	if len(samples) > 0 {
		resp.AvgLatencyMs = float64(total.Milliseconds()) / float64(len(samples))
	}
	return resp
}
