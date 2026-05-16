package models

type Product struct {
	Title       string  `json:"title"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	Rating      int     `json:"rating"`
	Reviews     int     `json:"reviews"`
}
