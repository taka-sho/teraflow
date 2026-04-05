package context

import (
	"regexp"
	"strings"
	"time"

	"github.com/taka-sho/teraflow/internal/index"
)

var nodeIDRE = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9_-]*:[a-zA-Z0-9_./-]+\b`)

// ScoreEntry scores one index entry for relevance to the current conversation.
func ScoreEntry(entry index.Entry, userInput, conversationHistory string) float64 {
	dependsOnScore := scoreDependsOn(entry, userInput)
	keywordScore := scoreKeywords(entry, userInput+"\n"+conversationHistory)
	recencyScore := scoreRecency(entry.UpdatedAt)
	return 0.4*dependsOnScore + 0.4*keywordScore + 0.2*recencyScore
}

func scoreDependsOn(entry index.Entry, userInput string) float64 {
	mentioned := extractNodeIDs(userInput)
	if len(mentioned) == 0 {
		return 0.2
	}
	if _, ok := mentioned[strings.ToLower(entry.NodeID)]; ok {
		return 1.0
	}

	for _, dep := range entry.DependsOn {
		if _, ok := mentioned[strings.ToLower(dep)]; ok {
			return 1.0
		}
		if sameNamespaceMentioned(dep, mentioned) {
			return 0.5
		}
	}

	if sameNamespaceMentioned(entry.NodeID, mentioned) {
		return 0.5
	}
	return 0.2
}

func scoreKeywords(entry index.Entry, text string) float64 {
	queryKeywords := extractKeywords(text)
	entryKeywords := extractKeywords(strings.ToLower(entry.Title + " " + strings.Join(entry.Tags, " ")))
	if len(queryKeywords) == 0 || len(entryKeywords) == 0 {
		return 0.0
	}

	matched := 0
	for kw := range entryKeywords {
		if _, ok := queryKeywords[kw]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(entryKeywords))
}

func scoreRecency(updatedAt time.Time) float64 {
	if updatedAt.IsZero() {
		return 0.3
	}
	age := time.Since(updatedAt)
	if age <= 7*24*time.Hour {
		return 1.0
	}
	if age <= 30*24*time.Hour {
		return 0.7
	}
	return 0.3
}

func extractNodeIDs(text string) map[string]struct{} {
	matches := nodeIDRE.FindAllString(strings.ToLower(text), -1)
	out := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		out[m] = struct{}{}
	}
	return out
}

func sameNamespaceMentioned(nodeID string, mentioned map[string]struct{}) bool {
	parts := strings.SplitN(strings.ToLower(nodeID), ":", 2)
	if len(parts) != 2 {
		return false
	}
	prefix := parts[0] + ":"
	for m := range mentioned {
		if strings.HasPrefix(m, prefix) {
			return true
		}
	}
	return false
}

func extractKeywords(text string) map[string]struct{} {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' && r < 0x80
	})

	out := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len([]rune(f)) < 2 {
			continue
		}
		out[f] = struct{}{}
	}
	return out
}
