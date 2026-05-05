package srv

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/acgh213/bookclub/db/dbgen"
)

// handleWheelAPI dispatches API requests for /api/round/{roundID}/...
func (s *Server) handleWheelAPI(w http.ResponseWriter, r *http.Request) {
	// Strip "/api/round/" prefix to get the path after it
	path := strings.TrimPrefix(r.URL.Path, "/api/round/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Round ID required", http.StatusBadRequest)
		return
	}

	// Parse round ID
	roundID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	// Check if there's a sub-path (like "wheel-books" or "spin")
	if len(parts) > 1 && parts[1] != "" {
		subPath := parts[1]
		if subPath == "wheel-books" && r.Method == http.MethodGet {
			s.handleWheelBooks(w, r, roundID)
			return
		}
		if subPath == "spin" && r.Method == http.MethodPost {
			s.handleSpinResult(w, r, roundID)
			return
		}
		if subPath == "set-current" && r.Method == http.MethodPost {
			s.handleSetCurrentBook(w, r, roundID)
			return
		}
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleWheelBooks returns books for the wheel, using round_entries if available,
// otherwise falling back to submissions.
func (s *Server) handleWheelBooks(w http.ResponseWriter, r *http.Request, roundID int64) {
	ctx := context.Background()

	// Try round_entries first
	entries, err := s.queries.ListRoundEntries(ctx, roundID)
	if err != nil {
		log.Printf("ListRoundEntries error: %v", err)
		http.Error(w, "Failed to load books", http.StatusInternalServerError)
		return
	}

	type bookJSON struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Author   string `json:"author"`
		CoverURL string `json:"cover_url"`
	}

	var books []bookJSON

	if len(entries) > 0 {
		// Use round_entries (library books)
		for _, entry := range entries {
			b := bookJSON{
				ID:     entry.ID,
				Title:  entry.Title,
				Author: entry.Author,
			}
			if entry.CoverUrl.Valid {
				b.CoverURL = entry.CoverUrl.String
			}
			books = append(books, b)
		}
	} else {
		// Fallback to submissions
		submissions, err := s.queries.ListSubmissionsByRound(ctx, roundID)
		if err != nil {
			log.Printf("ListSubmissionsByRound error: %v", err)
			http.Error(w, "Failed to load books", http.StatusInternalServerError)
			return
		}

		for _, sub := range submissions {
			books = append(books, bookJSON{
				ID:     sub.ID,
				Title:  sub.BookTitle,
				Author: sub.BookAuthor,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// handleSpinResult records a spin result (the winning book) in round_results.
func (s *Server) handleSpinResult(w http.ResponseWriter, r *http.Request, roundID int64) {
	ctx := context.Background()

	// Decode request body
	var req struct {
		BookID int64 `json:"book_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create round result
	result, err := s.queries.CreateRoundResult(ctx, dbgen.CreateRoundResultParams{
		RoundID:    roundID,
		BookID:     sql.NullInt64{Int64: req.BookID, Valid: true},
		ResultType: "spin",
		SelectedBy: sql.NullString{String: "wheel", Valid: true},
	})
	if err != nil {
		log.Printf("CreateRoundResult error: %v", err)
		http.Error(w, "Failed to record spin result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
