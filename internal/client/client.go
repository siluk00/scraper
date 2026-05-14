package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Product struct {
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Rating      int     `json:"rating"`
	Reviews     int     `json:"reviews"`
}

type ScraperClient struct {
	BaseURL string
}

// função escalável
func NewScraperClient(baseURL string) *ScraperClient {
	return &ScraperClient{BaseURL: baseURL}
}

func (c *ScraperClient) GetProducts(brand string) ([]Product, error) {
	// Por enquanto a API suporta apenas /lenovo fixo,
	// mas o cliente está preparado para expansão.
	url := fmt.Sprintf("%s/%s", c.BaseURL, brand)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}

	return products, nil
}
