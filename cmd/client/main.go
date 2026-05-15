package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/siluk00/scraper/internal/client"
)

func main() {
	brand := flag.String("brand", "lenovo", "Brand to search for")
	serverURL := flag.String("server", "http://localhost:8080", "Server URL")
	flag.Parse()

	c := client.NewScraperClient(*serverURL)
	products, err := c.GetProducts(*brand)
	if err != nil {
		log.Fatalf("Error fetching products: %v", err)
	}

	fmt.Printf("Encontrados %d notebooks %s:\n", len(products), *brand)
	for _, p := range products {
		fmt.Printf("- %s: $%.2f (%d reviews, %d rating)\n  URL: %s\n", p.Title, p.Price, p.Reviews, p.Rating, p.Url)
	}
}
