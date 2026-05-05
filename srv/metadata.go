package srv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

// BookMetadata is a search result from any metadata provider.
type BookMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	PageCount   int    `json:"page_count"`
	ISBN        string `json:"isbn"`
	SourceID    string `json:"source_id"`
	Source      string `json:"source"`
	Year        string `json:"year"`
}

// BookMetadataService searches for book metadata from external providers.
// Add new implementations (Open Library, Hardcover) behind this interface
// without changing the UI or import flow.
type BookMetadataService interface {
	SearchBooks(query string) ([]BookMetadata, error)
}

// --- Google Books implementation ---

type GoogleBooksService struct {
	apiKey string
	client *http.Client
}

func NewGoogleBooksService() *GoogleBooksService {
	return &GoogleBooksService{
		apiKey: os.Getenv("BOOKS_API_KEY"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *GoogleBooksService) SearchBooks(query string) ([]BookMetadata, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("BOOKS_API_KEY not set")
	}

	u := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&key=%s&maxResults=10",
		url.QueryEscape(query), s.apiKey)

	resp, err := s.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("google books request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books returned %d", resp.StatusCode)
	}

	var result googleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var books []BookMetadata
	for _, item := range result.Items {
		vol := item.VolumeInfo
		bm := BookMetadata{
			Title:       vol.Title,
			Description: vol.Description,
			PageCount:   vol.PageCount,
			Source:      "google_books",
			SourceID:    item.ID,
		}

		if len(vol.Authors) > 0 {
			bm.Author = vol.Authors[0]
		}

		if vol.ImageLinks != nil {
			bm.CoverURL = vol.ImageLinks.Thumbnail
			if bm.CoverURL == "" {
				bm.CoverURL = vol.ImageLinks.SmallThumbnail
			}
		}

		if len(vol.IndustryIdentifiers) > 0 {
			for _, id := range vol.IndustryIdentifiers {
				if id.Type == "ISBN_13" {
					bm.ISBN = id.Identifier
					break
				}
			}
			if bm.ISBN == "" {
				bm.ISBN = vol.IndustryIdentifiers[0].Identifier
			}
		}

		if vol.PublishedDate != "" {
			if len(vol.PublishedDate) >= 4 {
				bm.Year = vol.PublishedDate[:4]
			}
		}

		books = append(books, bm)
	}

	return books, nil
}

// --- Manual fallback (no API, just returns empty) ---

type ManualMetadataService struct{}

func (s *ManualMetadataService) SearchBooks(query string) ([]BookMetadata, error) {
	return nil, nil
}

// resolveMetadataService picks the metadata provider based on env.
func resolveMetadataService() BookMetadataService {
	if os.Getenv("BOOKS_API_KEY") != "" {
		return NewGoogleBooksService()
	}
	return &ManualMetadataService{}
}

// --- Google Books API response types ---

type googleBooksResponse struct {
	Items []googleBooksItem `json:"items"`
}

type googleBooksItem struct {
	ID         string             `json:"id"`
	VolumeInfo googleBooksVolume  `json:"volumeInfo"`
}

type googleBooksVolume struct {
	Title               string                    `json:"title"`
	Authors             []string                  `json:"authors"`
	Description         string                    `json:"description"`
	PageCount           int                       `json:"pageCount"`
	PublishedDate       string                    `json:"publishedDate"`
	ImageLinks          *googleBooksImageLinks    `json:"imageLinks"`
	IndustryIdentifiers []googleBooksIndustryID   `json:"industryIdentifiers"`
}

type googleBooksImageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
}

type googleBooksIndustryID struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

// --- Admin handler ---

func (s *Server) handleAdminLookupBook(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	results, err := s.metadata.SearchBooks(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Lookup failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
