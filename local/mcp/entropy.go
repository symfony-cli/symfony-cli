package mcp

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

func redactHighEntropy(str string) string {
	var result strings.Builder
	var word strings.Builder
	for i := 0; i < len(str); i++ {
		char := rune(str[i])
		if unicode.IsSpace(char) || char == '=' || char == ':' || char == ',' || char == ';' || char == '|' {
			if word.Len() > 0 {
				result.WriteString(doRedactHighEntropy(word.String()))
				word.Reset()
			}
			result.WriteRune(char)
		} else {
			word.WriteRune(char)
		}
	}
	if word.Len() > 0 {
		result.WriteString(doRedactHighEntropy(word.String()))
	}
	return result.String()
}

func doRedactHighEntropy(str string) string {
	if len(str) < 8 {
		return str
	}
	secretPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[A-Fa-f0-9]{32,}$`),                                                 // Hex
		regexp.MustCompile(`^[A-Za-z0-9+/]{32,}={0,2}$`),                                         // Base64
		regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`), // UUID (case insensitive)
	}
	for _, pattern := range secretPatterns {
		if pattern.MatchString(str) {
			return "[REDACTED]"
		}
	}
	freqMap := make(map[rune]float64)
	totalChars := float64(len(str))
	for _, char := range str {
		freqMap[char]++
	}
	entropy := 0.0
	for _, count := range freqMap {
		prob := count / totalChars
		entropy -= prob * math.Log2(prob)
	}
	threshold := 3.5
	if len(str) > 32 {
		threshold = 3.75
	} else if len(str) > 16 {
		threshold = 3.25
	}
	charSetScore := float64(len(freqMap)) / float64(len(str))
	if charSetScore > 0.5 {
		threshold -= 0.25
	}
	if entropy > threshold {
		return "[REDACTED]"
	}
	return str
}
