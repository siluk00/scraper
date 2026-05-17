package scraper

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/siluk00/scraper/internal/client"
	"github.com/siluk00/scraper/internal/models"
	"golang.org/x/net/html"
)

var cache = struct {
	mu        sync.RWMutex
	products  []models.Product
	fetchedAt time.Time
}{}

const cacheTTL = 5 * time.Minute

const (
	maxRetries     = 5
	baseDelay      = 1 * time.Second
	maxDelay       = 16 * time.Second
	jitterFraction = 0.3
	)

func fetchWithRetry(fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential base: 1s, 2s, 4s, 8s.
			delay := baseDelay * (1 << (attempt - 1))
			if delay > maxDelay {
				delay = maxDelay
			}
			// Full jitter: multiply by a random factor in [1-jitter, 1+jitter]
			jitter := 1 + (rand.Float64()*2-1)*jitterFraction
			sleep := time.Duration(float64(delay) * jitter)
			slog.Warn("retrying after backoff",
				"attempt", attempt,
				"sleep_ms", sleep.Milliseconds(),
				"error", lastErr,
			)
			time.Sleep(sleep)
		}

		resp, err := fn()
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 429 and 5xx (server errors).
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

func ScrapeProducts(httpClient *http.Client, baseURL string, brandFilter string) ([]models.Product, error) {
	// First verify cache
	cache.mu.RLock()
	if time.Since(cache.fetchedAt) < cacheTTL && len(cache.products) > 0 {
		cached := make([]models.Product, len(cache.products))
		copy(cached, cache.products)
		cache.mu.RUnlock()
		slog.Info("returning cached products", "count", len(cached), "age_s", int(time.Since(cache.fetchedAt).Seconds()))
		return cached, nil
	}
	cache.mu.RUnlock()

	var allProducts []models.Product
	page := 1

	slog.Info("starting scraper", "brand", brandFilter, "url", baseURL)

	for {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		slog.Debug("fetching page", "page", page, "url", url)

		productsOnPage, stop, err := func(targetURL string) ([]models.Product, bool, error) {
			req, err := http.NewRequest("GET", targetURL, nil)
			if err != nil {
				return nil, false, err
			}
			client.SetBrowserHeaders(req)

			resp, err := fetchWithRetry(func() (*http.Response, error) {
				return httpClient.Do(req)
			})
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

	sort.Slice(allProducts, func(i, j int) bool {
		return allProducts[i].Price < allProducts[j].Price
	})

	slog.Info("scraping completed", "total_products", len(allProducts))

	// Persist to cache.
	cache.mu.Lock()
	cache.products = allProducts
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()

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
			for _, c := range strings.Fields(a.Val) {
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
	fields := strings.Fields(text)
	if len(fields) > 0 {
		count, _ := strconv.Atoi(fields[0])
		return count
	}
	return 0
}

func extractRating(n *html.Node) int {
	var rating int
	var f func(*html.Node)
	f = func(m *html.Node) {
		if m.Type == html.ElementNode && m.Data == "p" {
			if val := getAttr(m, "data-rating"); val != "" {
				rating, _ = strconv.Atoi(val)
				return
			}
		}
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

// ScrapeLenovo keeps backward compatibility.
func ScrapeLenovo(baseURL string) ([]models.Product, error) {
	return ScrapeProducts(client.BrowserClient, baseURL, "lenovo")
}
