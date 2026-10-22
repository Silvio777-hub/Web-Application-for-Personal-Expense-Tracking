package ocr

import (
	"context"
	"math/rand"
	"time"
)

// OCRResult represents the data extracted from a receipt.
type OCRResult struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Confidence  float64   `json:"confidence"`
	Engine      string    `json:"engine"`
}

// Processor defines the interface for OCR engines.
type Processor interface {
	Process(ctx context.Context, filePath string) (*OCRResult, error)
}

// MockProcessor simulates an AI OCR engine with realistic variations.
type MockProcessor struct {
	ErrorRate float64 // Probability of returning an error
}

func (p *MockProcessor) Process(ctx context.Context, filePath string) (*OCRResult, error) {
	// Simulate variable processing time (latency)
	latency := 1000 + rand.Intn(2000)
	time.Sleep(time.Duration(latency) * time.Millisecond)

	// Randomly fail based on ErrorRate
	if rand.Float64() < p.ErrorRate {
		return nil, context.DeadlineExceeded // Simulate a typical OCR timeout/failure
	}

	// Randomized Confidence
	confidence := 0.85 + (rand.Float64() * 0.14) // 85% - 99%

	// Return randomized but "sensible" demo data
	return &OCRResult{
		Amount:      42.50 + float64(rand.Intn(100)),
		Currency:    "USD",
		Description: "Simulated Automated Extraction",
		Date:        time.Now(),
		Confidence:  confidence,
		Engine:      "MockAI-v2",
	}, nil
}
