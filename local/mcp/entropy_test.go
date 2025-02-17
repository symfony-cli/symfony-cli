package mcp

import "testing"

func TestAnalyzeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Base64 Token", "Some token eyJhbGciOiJIUzI1NiIs detected", "Some token [REDACTED] detected"},
		{"Base64 Token =", "Some_token=eyJhbGciOiJIUzI1NiIs detected", "Some_token=[REDACTED] detected"},
		{"UUID", "A UUID 550e8400-e29b-41d4-a716-446655440000", "A UUID [REDACTED]"},
		{"Random", "Random aB1$x9#mK2&pL5@vN8*qR3", "Random [REDACTED]"},
		{"AWS Secret", "aws_secret_key=AKIA4YFAKESECRETKEY123EXAMPLE", "aws_secret_key=[REDACTED]"},
		{"AWS Secret in text", "The key AKIA4YFAKESECRETKEY123EXAMPLE was exposed", "The key [REDACTED] was exposed"},
		{"Stripe Secret Key", "stripe_key=sk_live_51HCOXXAaYYbbiuYYuu990011", "stripe_key=[REDACTED]"},
		{"Stripe Key", "sk_live_h9xj4h44j3h43jh43", "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactHighEntropy(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
