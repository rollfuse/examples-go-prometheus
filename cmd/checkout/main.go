// Command checkout is a minimal e-commerce checkout service demonstrating
// @rollfuse/go-sdk instrumented with Prometheus: every request evaluates
// the "checkout-redesign" flag locally (no network call on the hot path,
// per ADR 0004 — see the SDK's own README) and records the outcome as
// Prometheus metrics, so a rollout's effect is visible on a dashboard in
// real time.
package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	rollfuse "github.com/rollfuse/go-sdk"
)

const flagKey = "checkout-redesign"

var (
	flagEvaluations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rollfuse_flag_evaluations_total",
		Help: "Total flag evaluations performed by this service, by flag, variation and reason.",
	}, []string{"flag_key", "variation_key", "reason"})

	checkoutDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "checkout_request_duration_seconds",
		Help:    "Simulated checkout processing time, by the variation the request was served.",
		Buckets: prometheus.DefBuckets,
	}, []string{"variation_key"})

	checkoutTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "checkout_requests_total",
		Help: "Total checkout requests, by the variation the request was served and its outcome.",
	}, []string{"variation_key", "outcome"})
)

func main() {
	baseURL := envOr("ROLLFUSE_API_BASE_URL", "http://localhost:8090")
	credential := envOr("ROLLFUSE_SERVICE_CREDENTIAL", "demo-credential")
	listenAddr := envOr("LISTEN_ADDR", ":8080")

	client, err := rollfuse.NewClient(baseURL, credential)
	if err != nil {
		log.Fatalf("rollfuse.NewClient: %v", err)
	}
	defer client.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Start(startCtx); err != nil {
		log.Fatalf("client.Start: %v (is the rollfuse API at %s reachable?)", err, baseURL)
	}

	log.Printf("checkout: connected to %s, serving on %s", baseURL, listenAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/checkout", checkoutHandler(client))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: listenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

type checkoutResponse struct {
	SubjectKey   string `json:"subject_key"`
	VariationKey string `json:"variation_key"`
	Reason       string `json:"reason"`
	Outcome      string `json:"outcome"`
}

// checkoutHandler evaluates the flag for the request's subject and
// simulates two different checkout implementations: "on" (the redesigned,
// faster flow) and everything else (the legacy flow), so the two
// variations produce visibly different latency distributions on the
// Prometheus histogram.
func checkoutHandler(client *rollfuse.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subjectKey := r.URL.Query().Get("user")
		if subjectKey == "" {
			subjectKey = randomSubjectKey()
		}

		result, err := client.Evaluate(subjectKey, flagKey, rollfuse.WithFallback(false))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		flagEvaluations.WithLabelValues(flagKey, result.VariationKey, string(result.Reason)).Inc()

		var enabled bool
		_ = json.Unmarshal(result.Value, &enabled)

		start := time.Now()
		outcome := simulateCheckout(enabled)
		checkoutDuration.WithLabelValues(result.VariationKey).Observe(time.Since(start).Seconds())
		checkoutTotal.WithLabelValues(result.VariationKey, outcome).Inc()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(checkoutResponse{
			SubjectKey:   subjectKey,
			VariationKey: result.VariationKey,
			Reason:       string(result.Reason),
			Outcome:      outcome,
		})
	}
}

// simulateCheckout stands in for the two real checkout implementations a
// production service would branch between. The redesigned flow ("on") is
// modeled as faster and slightly less error-prone, purely so the demo's
// Prometheus dashboard has something visually interesting to show.
func simulateCheckout(redesigned bool) string {
	if redesigned {
		time.Sleep(time.Duration(20+rand.IntN(30)) * time.Millisecond)

		if rand.IntN(100) < 2 {
			return "error"
		}

		return "success"
	}

	time.Sleep(time.Duration(80+rand.IntN(120)) * time.Millisecond)

	if rand.IntN(100) < 5 {
		return "error"
	}

	return "success"
}

func randomSubjectKey() string {
	return "user_" + strconv.Itoa(rand.IntN(10_000))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
