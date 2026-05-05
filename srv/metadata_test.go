package srv

import (
	"encoding/json"
	"testing"
)

func TestGoogleBooksParse(t *testing.T) {
	// Simulated Google Books API response
	const responseJSON = `{
		"items": [
			{
				"id": "abc123",
				"volumeInfo": {
					"title": "Project Hail Mary",
					"authors": ["Andy Weir"],
					"description": "A lone astronaut must save humanity.",
					"pageCount": 496,
					"publishedDate": "2021-05-04",
					"imageLinks": {
						"smallThumbnail": "http://example.com/small.jpg",
						"thumbnail": "http://example.com/thumb.jpg"
					},
					"industryIdentifiers": [
						{"type": "ISBN_13", "identifier": "9780593135204"},
						{"type": "ISBN_10", "identifier": "0593135202"}
					]
				}
			},
			{
				"id": "def456",
				"volumeInfo": {
					"title": "No Cover Book",
					"authors": [],
					"industryIdentifiers": [
						{"type": "OTHER", "identifier": "X-123"}
					]
				}
			}
		]
	}`

	var resp googleBooksResponse
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	// First item — full data
	item1 := resp.Items[0]
	if item1.VolumeInfo.Title != "Project Hail Mary" {
		t.Errorf("title: got %q", item1.VolumeInfo.Title)
	}
	if len(item1.VolumeInfo.Authors) != 1 || item1.VolumeInfo.Authors[0] != "Andy Weir" {
		t.Errorf("authors: got %v", item1.VolumeInfo.Authors)
	}
	if item1.VolumeInfo.PageCount != 496 {
		t.Errorf("pageCount: got %d", item1.VolumeInfo.PageCount)
	}
	if item1.VolumeInfo.ImageLinks == nil || item1.VolumeInfo.ImageLinks.Thumbnail != "http://example.com/thumb.jpg" {
		t.Errorf("imageLinks: got %v", item1.VolumeInfo.ImageLinks)
	}

	// ISBN_13 preferred
	isbn13 := ""
	for _, id := range item1.VolumeInfo.IndustryIdentifiers {
		if id.Type == "ISBN_13" {
			isbn13 = id.Identifier
		}
	}
	if isbn13 != "9780593135204" {
		t.Errorf("isbn13: got %q", isbn13)
	}

	// Second item — edge case (no authors, no images, non-ISBN identifier)
	item2 := resp.Items[1]
	if item2.VolumeInfo.Title != "No Cover Book" {
		t.Errorf("title: got %q", item2.VolumeInfo.Title)
	}
	if len(item2.VolumeInfo.Authors) != 0 {
		t.Errorf("authors: expected empty, got %v", item2.VolumeInfo.Authors)
	}
	if item2.VolumeInfo.ImageLinks != nil {
		t.Errorf("imageLinks: expected nil")
	}

	// Test BookMetadata conversion
	svc := &GoogleBooksService{}
	// We can't call SearchBooks without an API key, but we've verified the parse
	_ = svc
}

func TestManualMetadataService(t *testing.T) {
	svc := &ManualMetadataService{}
	results, err := svc.SearchBooks("anything")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestResolveMetadataService(t *testing.T) {
	// Without BOOKS_API_KEY, should return ManualMetadataService
	t.Setenv("BOOKS_API_KEY", "")
	svc := resolveMetadataService()
	if _, ok := svc.(*ManualMetadataService); !ok {
		t.Errorf("expected ManualMetadataService without API key, got %T", svc)
	}
}
