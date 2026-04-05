package context

import (
	"regexp"
	"unicode"
)

var englishWordRE = regexp.MustCompile(`[A-Za-z0-9_]+`)

// EstimateTokens approximates token count for mixed Japanese/English text.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	chars := 0
	hasJapanese := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		chars++
		if isJapaneseRune(r) {
			hasJapanese = true
		}
	}

	wordCount := len(englishWordRE.FindAllString(text, -1))
	hasEnglish := wordCount > 0

	jpEstimate := float64(chars) * 0.5
	enEstimate := float64(wordCount) * 1.3

	switch {
	case hasJapanese && hasEnglish:
		return int((jpEstimate + enEstimate) / 2.0)
	case hasJapanese:
		return int(jpEstimate)
	case hasEnglish:
		return int(enEstimate)
	default:
		return int(jpEstimate)
	}
}

func isJapaneseRune(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han)
}
