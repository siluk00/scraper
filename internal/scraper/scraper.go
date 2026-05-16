package scraper

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/siluk00/scraper/internal/client"
	"github.com/siluk00/scraper/internal/models"
	"golang.org/x/net/html"
)

func ScrapeProducts(httpClient *http.Client, baseURL string, brandFilter string) ([]models.Product, error) {
	var allProducts []models.Product
	page := 1

	slog.Info("starting scraper", "brand", brandFilter, "url", baseURL)

	// main loop for pagination
	for {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		slog.Debug("fetching page", "page", page, "url", url)

		productsOnPage, stop, err := func(targetURL string) ([]models.Product, bool, error) {
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				return nil, false, err
			}

			client.SetBrowserHeaders(req)

			resp, err := httpClient.Do(req)
			if err != nil {
				return nil, false, err
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				slog.Warn("pagination stopped", "page", page, "status", resp.StatusCode)
				return nil, true, nil
			}

			doc, err := html.Parse(resp.Body)
			if err != nil {
				return nil, false, err
			}

			// recursive function to traverse the html tree nodes
			var pageProducts []models.Product
			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.ElementNode && hasClass(n, "thumbnail") {
					p := extractProduct(n)
					if brandMatch(p, brandFilter) {
						pageProducts = append(pageProducts, p)
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			f(doc)

			slog.Info("page processed", "page", page, "matches", len(pageProducts))
			return pageProducts, len(pageProducts) == 0, nil
		}(url)

		if err != nil {
			slog.Error("scrape failed", "page", page, "error", err)
			return nil, err
		}
		if stop {
			break
		}

		allProducts = append(allProducts, productsOnPage...)
		page++
	}

	slog.Info("scraping completed", "total_products", len(allProducts))
	sort.Slice(allProducts, func(i, j int) bool {
		return allProducts[i].Price < allProducts[j].Price
	})
	return allProducts, nil
}

func brandMatch(p models.Product, brand string) bool {
	if brand == "" {
		return true
	}
	titleLower := strings.ToLower(p.Title)
	descLower := strings.ToLower(p.Description)
	brandLower := strings.ToLower(brand)

	if strings.Contains(titleLower, brandLower) || strings.Contains(descLower, brandLower) {
		return true
	}

	// Special logic for Lenovo
	if brandLower == "lenovo" {
		if strings.Contains(titleLower, "thinkpad") || strings.Contains(titleLower, "ideapad") {
			return true
		}
	}
	return false
}

func hasClass(n *html.Node, className string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			classes := strings.Fields(a.Val)
			for _, c := range classes {
				if c == className {
					return true
				}
			}
		}
	}
	return false
}

func extractProduct(n *html.Node) models.Product {
	var p models.Product
	var f func(*html.Node)
	f = func(m *html.Node) {
		if m.Type == html.ElementNode {
			if m.Data == "a" && hasClass(m, "title") {
				p.Title = getAttr(m, "title")
				if p.Title == "" {
					p.Title = getText(m)
				}
				p.URL = "https://webscraper.io" + getAttr(m, "href")
			}
			if hasClass(m, "price") {
				priceStr := strings.TrimPrefix(getText(m), "$")
				p.Price, _ = strconv.ParseFloat(priceStr, 64)
			}
			if hasClass(m, "description") {
				p.Description = getText(m)
			}
			if hasClass(m, "ratings") {
				// Extract reviews and rating
				p.Reviews = extractReviews(m)
				p.Rating = extractRating(m)
			}
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return p
}

func extractReviews(n *html.Node) int {
	text := getText(n)
	// Example: "4 reviews"
	fields := strings.Fields(text)
	if len(fields) > 0 {
		count, _ := strconv.Atoi(fields[0])
		return count
	}
	return 0
}

func extractRating(n *html.Node) int {
	// Rating is usually in spans with stars
	// <p data-rating="3">...
	var rating int
	var f func(*html.Node)
	f = func(m *html.Node) {
		if m.Type == html.ElementNode && m.Data == "p" {
			if val := getAttr(m, "data-rating"); val != "" {
				rating, _ = strconv.Atoi(val)
				return
			}
		}
		// Alternative: count spans with "glyphicon-star" class
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}

	}
	f(n)
	return rating
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func getText(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(m *html.Node) {
		if m.Type == html.TextNode {
			b.WriteString(m.Data)
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(b.String())
}

// Keep compatibility
func ScrapeLenovo(baseURL string) ([]models.Product, error) {
	return ScrapeProducts(client.BrowserClient, baseURL, "lenovo")
}
