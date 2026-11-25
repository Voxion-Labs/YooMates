package search

import (
	"encoding/json"
	"testing"
)

func TestMatchItem_ExactAndPartial(t *testing.T) {
	msg := SearchMessage{
		ID:         "msg_1",
		AuthorName: "Rudranarayan Jena",
		Content:    "The new Go architecture runs with zero node_modules and zero TypeScript runtime overhead.",
		FileName:   "architecture.png",
	}

	// 1. Content partial match
	matched, score, snippet := MatchItem("node_modules", msg)
	if !matched {
		t.Errorf("expected 'node_modules' to match")
	}
	if score <= 0 {
		t.Errorf("expected score > 0, got %d", score)
	}
	if snippet == "" {
		t.Errorf("expected non-empty snippet")
	}

	// 2. Author match
	matched, score, _ = MatchItem("Rudranarayan", msg)
	if !matched || score <= 0 {
		t.Errorf("expected author match")
	}

	// 3. File attachment name match
	matched, score, _ = MatchItem("architecture.png", msg)
	if !matched || score <= 0 {
		t.Errorf("expected attachment name match")
	}

	// 4. Non-matching query
	matched, _, _ = MatchItem("nonexistent_keyword_xyz", msg)
	if matched {
		t.Errorf("expected non-matching query to return false")
	}

	// 5. Empty query
	matched, _, _ = MatchItem("", msg)
	if matched {
		t.Errorf("expected empty query to return false")
	}
}

func TestExecuteSearchJSON(t *testing.T) {
	messages := []SearchMessage{
		{ID: "m1", AuthorName: "Alice", Content: "Hello world from Go!"},
		{ID: "m2", AuthorName: "Bob", Content: "WebAssembly provides zero-latency client search."},
		{ID: "m3", AuthorName: "Charlie", Content: "MySQL 8.4 InnoDB schema migrations."},
	}

	rawJSON, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("failed to marshal messages: %v", err)
	}

	// 1. Search for WebAssembly
	resJSON, err := ExecuteSearchJSON("WebAssembly", string(rawJSON))
	if err != nil {
		t.Fatalf("ExecuteSearchJSON failed: %v", err)
	}

	var res SearchResponse
	if err := json.Unmarshal([]byte(resJSON), &res); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	if res.Total != 1 {
		t.Fatalf("expected 1 result, got %d", res.Total)
	}
	if res.Results[0].ID != "m2" {
		t.Errorf("expected match ID 'm2', got %s", res.Results[0].ID)
	}

	// 2. Search empty query
	emptyResJSON, err := ExecuteSearchJSON("", string(rawJSON))
	if err != nil {
		t.Fatalf("ExecuteSearchJSON empty query failed: %v", err)
	}
	var emptyRes SearchResponse
	_ = json.Unmarshal([]byte(emptyResJSON), &emptyRes)
	if emptyRes.Total != 0 {
		t.Errorf("expected 0 results for empty query, got %d", emptyRes.Total)
	}
}
