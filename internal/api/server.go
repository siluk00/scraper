package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/siluk00/scraper/internal/scraper"
)

func StartServer(port string) {
	http.HandleFunc("/lenovo", lenovoHandler)

	log.Printf("Server starting on %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func lenovoHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := "https://webscraper.io/test-sites/e-commerce/static/computers/laptops"
	
	products, err := scraper.ScrapeLenovo(baseURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}
