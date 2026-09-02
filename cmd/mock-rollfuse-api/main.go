// Command mock-rollfuse-api is a minimal stand-in for the real rollfuse
// platform API, so this example runs end-to-end with `docker-compose up`
// alone — no rollfuse account or credential required. It serves a static
// Configuration for GET /v1/config (see config/checkout-config.json) and
// accepts (and logs) POST /v1/exposure-events, matching the wire shapes
// github.com/rollfuse/go-sdk expects.
//
// This is a demo fixture, not a reference implementation of the rollfuse
// API: it does not validate the Authorization header, version
// Configuration, or persist anything.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	configPath := envOr("CONFIG_PATH", "/config/checkout-config.json")
	listenAddr := envOr("LISTEN_ADDR", ":8090")

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("reading %s: %v", configPath, err)
	}

	// Fail fast if the fixture isn't even valid JSON — this file is
	// hand-maintained, not generated.
	var probe map[string]any
	if err := json.Unmarshal(configBytes, &probe); err != nil {
		log.Fatalf("%s is not valid JSON: %v", configPath, err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(configBytes)
	})

	mux.HandleFunc("POST /v1/exposure-events", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		var batch struct {
			Events []json.RawMessage `json:"events"`
		}
		if err := json.Unmarshal(body, &batch); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)

			return
		}

		log.Printf("exposure-events: accepted %d event(s)", len(batch.Events))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]int{"accepted": len(batch.Events)})
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("mock-rollfuse-api: serving %s on %s", configPath, listenAddr)

	if err := http.ListenAndServe(listenAddr, mux); err != nil { //nolint:gosec // demo-only fixture, no external exposure
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}
