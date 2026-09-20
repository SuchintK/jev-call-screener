package domain

// Category is a provider-independent classification of a caller's purpose.
type Category string

const (
	CategoryPromotional Category = "promotional"
	CategoryWanted      Category = "wanted"
	CategoryUnclear     Category = "unclear"
)

// Valid reports whether c is a supported category.
func (c Category) Valid() bool {
	return c == CategoryPromotional || c == CategoryWanted || c == CategoryUnclear
}

// Classification is the calibrated result returned by a classifier.
type Classification struct {
	Category      Category             `json:"category"`
	Confidence    float64              `json:"confidence"`
	Probabilities map[Category]float64 `json:"probabilities,omitempty"`
	Model         string               `json:"model,omitempty"`
}
