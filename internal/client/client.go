package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/siluk00/scraper/internal/models"
)

type ScraperClient struct {
	BaseURL string
}

// função escalável
func NewScraperClient(baseURL string) *ScraperClient {
	return &ScraperClient{BaseURL: baseURL}
}

func (c *ScraperClient) GetProducts(brand string) ([]models.Product, error) {
	// Por enquanto a API suporta apenas /lenovo fixo,
	// mas o cliente está preparado para expansão.
	url := fmt.Sprintf("%s/%s", c.BaseURL, brand)
	httpClient := &http.Client{Timeout: 10 * time.Second}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var products []models.Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}

	return products, nil
}
