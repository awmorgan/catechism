package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func resolvePath(rel string) string {
	if _, err := os.Stat(rel); err == nil {
		return rel
	}
	parentRel := filepath.Join("../..", rel)
	if _, err := os.Stat(parentRel); err == nil {
		return parentRel
	}
	return rel
}

// TestParagraphCoverage verifies that all 2,865 paragraphs exist in para-map.json
// and point to valid generated page files.
func TestParagraphCoverage(t *testing.T) {
	data, err := os.ReadFile(resolvePath("assets/para-map.json"))
	if err != nil {
		t.Fatalf("Failed to read assets/para-map.json: %v", err)
	}

	var paraMap map[string]string
	if err := json.Unmarshal(data, &paraMap); err != nil {
		t.Fatalf("Failed to parse assets/para-map.json: %v", err)
	}

	if len(paraMap) != 2865 {
		t.Errorf("Expected 2,865 mapped paragraphs, got %d", len(paraMap))
	}

	for i := 1; i <= 2865; i++ {
		key := fmt.Sprintf("%d", i)
		target, ok := paraMap[key]
		if !ok {
			t.Errorf("Missing paragraph ¶ %d in para-map.json", i)
			continue
		}
		if _, err := os.Stat(resolvePath(target)); os.IsNotExist(err) {
			t.Errorf("Paragraph ¶ %d points to non-existent file: %s", i, target)
		}
	}
}

// TestSearchIndex verifies that the search index contains all paragraphs.
func TestSearchIndex(t *testing.T) {
	data, err := os.ReadFile(resolvePath("assets/search-index.json"))
	if err != nil {
		t.Fatalf("Failed to read assets/search-index.json: %v", err)
	}

	var items []SearchItem
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("Failed to parse assets/search-index.json: %v", err)
	}

	if len(items) != 2865 {
		t.Errorf("Expected 2,865 search items, got %d", len(items))
	}
}

// TestPageBoundaries verifies that the Prologue strictly ends at ¶ 25 on page 1,
// and Part One begins with ¶ 26 on page 2.
func TestPageBoundaries(t *testing.T) {
	data, err := os.ReadFile(resolvePath("assets/para-map.json"))
	if err != nil {
		t.Fatalf("Failed to read assets/para-map.json: %v", err)
	}

	var paraMap map[string]string
	if err := json.Unmarshal(data, &paraMap); err != nil {
		t.Fatalf("Failed to parse assets/para-map.json: %v", err)
	}

	for p := 1; p <= 25; p++ {
		key := fmt.Sprintf("%d", p)
		if target := paraMap[key]; target != "pages/001.html" {
			t.Errorf("Expected ¶ %d to be in pages/001.html, got %s", p, target)
		}
	}

	for p := 26; p <= 49; p++ {
		key := fmt.Sprintf("%d", p)
		if target := paraMap[key]; target != "pages/002.html" {
			t.Errorf("Expected ¶ %d to be in pages/002.html, got %s", p, target)
		}
	}
}
