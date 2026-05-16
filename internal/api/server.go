package api

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

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
	http.HandleFunc("/healthz", healthHandler)

	slog.Info("server starting", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func lenovoHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("handling request", "path", r.URL.Path, "remote_addr", r.RemoteAddr)

	products, err := scraper.ScrapeProducts(targetURL, "lenovo")
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
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

	if err != nil || resp.StatusCode >= 500 {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = http.StatusText(resp.StatusCode)
			_ = resp.Body.Close()
		}

		slog.Warn("healthz degraded", "error", errMsg, "latency_ms", latency)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:          "degraded",
			TargetReachable: false,
			LatencyMs:       latency,
			Error:           errMsg,
		})
		return
	}
	_ = resp.Body.Close()

	slog.Debug("healthz ok", "latency_ms", latency)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:          "ok",
		TargetReachable: true,
		LatencyMs:       latency,
	})
}
