package api

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/siluk00/scraper/internal/scraper"
)

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
	baseURL := "https://webscraper.io/test-sites/e-commerce/static/computers/laptops"

	slog.Info("handling request", "path", r.URL.Path, "remote_addr", r.RemoteAddr)

	products, err := scraper.ScrapeProducts(baseURL, "lenovo")
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
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
