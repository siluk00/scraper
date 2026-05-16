package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siluk00/scraper/internal/models"
)

func TestServerReturnsCorrectLenovo(t *testing.T) {
	// Mock API response
	expectedNotebook := models.Product{
		Title:       "Lenovo ThinkPad L570",
		Price:       999.00,
		Description: "Lenovo ThinkPad L570, 15.6\" FHD, Core i7-7500U, 8GB, 256GB SSD, Windows 10 Pro",
		URL:         "https://webscraper.io/test-product",
		Rating:      3,
		Reviews:     11,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/lenovo" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]models.Product{expectedNotebook}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Using standard Go client to call the mocked server
	resp, err := http.Get(server.URL + "/lenovo")
	if err != nil {
		t.Fatalf("Failed to call server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", resp.Status)
	}

	var products []models.Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(products) == 0 {
		t.Fatal("Expected at least one product, got none")
	}

	found := false
	for _, p := range products {
		if p.Title == "Lenovo ThinkPad L570" {
			found = true
			if p.Price != 999.00 {
				t.Errorf("Expected price 999, got %f", p.Price)
			}
			if p.Rating != 3 {
				t.Errorf("Expected rating 3, got %d", p.Rating)
			}
			if p.Reviews != 11 {
				t.Errorf("Expected 11 reviews, got %d", p.Reviews)
			}
			expectedDesc := "Lenovo ThinkPad L570, 15.6\" FHD, Core i7-7500U, 8GB, 256GB SSD, Windows 10 Pro"
			if p.Description != expectedDesc {
				t.Errorf("Description mismatch")
			}
			if p.URL != "https://webscraper.io/test-product" {
				t.Errorf("Expected URL https://webscraper.io/test-product, got %s", p.URL)
			}
		}
	}

	if !found {
		t.Error("Lenovo ThinkPad L570 was not found in the client response")
	}
}
