package search

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SearchMessage represents a searchable message entity
type SearchMessage struct {
	ID         string `json:"id"`
	AuthorName string `json:"author_name"`
	Content    string `json:"content"`
	FileName   string `json:"file_name,omitempty"`
}

// SearchResult represents a scored search match
type SearchResult struct {
	ID        string `json:"id"`
	Score     int    `json:"score"`
	MatchedOn string `json:"matched_on"`
	Snippet   string `json:"snippet"`
}

// SearchResponse is returned to the client
type SearchResponse struct {
	Query   string         `json:"query"`
	Total   int            `json:"total"`
	Results []SearchResult `json:"results"`
}

// MatchItem checks whether a message matches the search query and calculates relevance score
func MatchItem(query string, msg SearchMessage) (bool, int, string) {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return false, 0, ""
	}

	contentLower := strings.ToLower(msg.Content)
	authorLower := strings.ToLower(msg.AuthorName)
	fileLower := strings.ToLower(msg.FileName)

	score := 0
	matchedOn := ""
	snippet := msg.Content

	// Exact author match
	if authorLower == q {
		score += 50
		matchedOn = "author (exact)"
	} else if strings.Contains(authorLower, q) {
		score += 25
		matchedOn = "author"
	}

	// File name match
	if fileLower != "" && strings.Contains(fileLower, q) {
		score += 30
		if matchedOn == "" {
			matchedOn = "attachment: " + msg.FileName
		}
	}

	// Content matching
	if contentLower == q {
		score += 100
		matchedOn = "content (exact)"
	} else if strings.Contains(contentLower, q) {
		score += 40
		if matchedOn == "" {
			matchedOn = "content"
		}

		// Extract a contextual snippet around the matched query
		idx := strings.Index(contentLower, q)
		start := idx - 20
		if start < 0 {
			start = 0
		}
		end := idx + len(q) + 30
		if end > len(msg.Content) {
			end = len(msg.Content)
		}

		snippet = msg.Content[start:end]
		if start > 0 {
			snippet = "..." + snippet
		}
		if end < len(msg.Content) {
			snippet = snippet + "..."
		}
	} else {
		// Word token matching
		tokens := strings.Fields(q)
		allMatched := true
		for _, token := range tokens {
			if !strings.Contains(contentLower, token) && !strings.Contains(authorLower, token) {
				allMatched = false
				break
			}
		}
		if allMatched && len(tokens) > 1 {
			score += 15 * len(tokens)
			if matchedOn == "" {
				matchedOn = "multi-term"
			}
		}
	}

	if score > 0 {
		return true, score, snippet
	}
	return false, 0, ""
}

// FilterMessages searches a slice of SearchMessage and returns matching results
func FilterMessages(query string, messages []SearchMessage) []SearchResult {
	var results []SearchResult
	for _, msg := range messages {
		matched, score, snippet := MatchItem(query, msg)
		if matched {
			results = append(results, SearchResult{
				ID:      msg.ID,
				Score:   score,
				Snippet: snippet,
			})
		}
	}
	return results
}

// ExecuteSearchJSON accepts query and a JSON array of messages, returning a JSON string result
func ExecuteSearchJSON(query string, messagesJSON string) (string, error) {
	if strings.TrimSpace(query) == "" {
		emptyRes := SearchResponse{Query: query, Total: 0, Results: []SearchResult{}}
		b, _ := json.Marshal(emptyRes)
		return string(b), nil
	}

	var messages []SearchMessage
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return "", fmt.Errorf("invalid messages json payload: %w", err)
	}

	results := FilterMessages(query, messages)
	res := SearchResponse{
		Query:   query,
		Total:   len(results),
		Results: results,
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("failed to marshal search response: %w", err)
	}
	return string(bytes), nil
}
