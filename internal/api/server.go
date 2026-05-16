package api

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/siluk00/scraper/internal/client"
	"github.com/siluk00/scraper/internal/scraper"
)

const targetURL = "https://webscraper.io/test-sites/e-commerce/static/computers/laptops"

func StartServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	http.HandleFunc("/lenovo", lenovoHandler)
	http.HandleFunc("/healthz", healthzHandler)

	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func lenovoHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("handling request", "path", r.URL.Path, "remote_addr", r.RemoteAddr)

	products, err := scraper.ScrapeProducts(client.BrowserClient, targetURL, "lenovo")
	if err != nil {
		slog.Error("request failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(products); err != nil {
		slog.Error("failed to encode products", "error", err)
	}
	slog.Info("request completed", "matches", len(products))
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	type healthResponse struct {
		Status          string `json:"status"`
		TargetReachable bool   `json:"target_reachable"`
		LatencyMs       int64  `json:"latency_ms"`
		Error           string `json:"error,omitempty"`
	}

	// Test if the target site can be reached with a short timeout
	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Head(targetURL) // HEAD - only checks connectivity, does not download body
	latency := time.Since(start).Milliseconds()

	w.Header().Set("Content-Type", "application/json")

	// 1. Handle network/connectivity errors
	if err != nil {
		slog.Warn("healthz degraded (network error)", "error", err, "latency_ms", latency)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:          "degraded",
			TargetReachable: false,
			LatencyMs:       latency,
			Error:           err.Error(),
		})
		return
	}

	// At this point, resp is guaranteed to be non-nil
	defer func() { _ = resp.Body.Close() }()

	// 2. Handle server-side errors from the target
	if resp.StatusCode >= 500 {
		errMsg := http.StatusText(resp.StatusCode)
		slog.Warn("healthz degraded (target server error)", "status", resp.StatusCode, "latency_ms", latency)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:          "degraded",
			TargetReachable: false,
			LatencyMs:       latency,
			Error:           errMsg,
		})
		return
	}

	// 3. Success case
	slog.Debug("healthz ok", "latency_ms", latency)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:          "ok",
		TargetReachable: true,
		LatencyMs:       latency,
	})
}
