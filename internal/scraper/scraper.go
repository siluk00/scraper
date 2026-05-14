package scraper

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Product struct {
	Title       string `json:"title"`
	Price       string `json:"price"`
	Description string `json:"description"`
}

func ScrapeLenovo(baseURL string) ([]Product, error) {
	var allProducts []Product
	page := 1

	for {
		url := fmt.Sprintf("%s?page=%d", baseURL, page)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break // Assume we reached the end of pages
		}

		doc, err := html.Parse(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		foundOnPage := 0
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && hasClass(n, "thumbnail") {
				p := extractProduct(n)
				// Match Lenovo, ThinkPad or IdeaPad which are Lenovo brands
				titleLower := strings.ToLower(p.Title)
				descLower := strings.ToLower(p.Description)
				if strings.Contains(titleLower, "lenovo") ||
					strings.Contains(titleLower, "thinkpad") ||
					strings.Contains(titleLower, "ideapad") ||
					strings.Contains(descLower, "lenovo") {
					allProducts = append(allProducts, p)
				}
				foundOnPage++
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)

		if foundOnPage == 0 {
			break
		}
		page++
	}

	return allProducts, nil
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

func extractProduct(n *html.Node) Product {
	var p Product
	var f func(*html.Node)
	f = func(m *html.Node) {
		if m.Type == html.ElementNode {
			if m.Data == "a" && hasClass(m, "title") {
				p.Title = getAttr(m, "title")
				if p.Title == "" {
					p.Title = getText(m)
				}
			}
			if hasClass(m, "price") {
				p.Price = getText(m)
			}
			if hasClass(m, "description") {
				p.Description = getText(m)
			}
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return p
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
