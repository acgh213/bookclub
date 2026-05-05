package srv

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/acgh213/bookclub/db/dbgen"
)

// handleAdminRoundBooks shows the round's book pool and manages entries.
func (s *Server) handleAdminRoundBooks(w http.ResponseWriter, r *http.Request, roundIDStr string) {
	ctx := r.Context()
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	round, err := s.queries.GetRound(ctx, roundID)
	if err != nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	// Books already in this round
	entries, _ := s.queries.ListRoundEntries(ctx, roundID)

	// All library books (for the add picker)
	allBooks, _ := s.queries.ListBooks(ctx)

	// Build set of book IDs already in round
	inRound := make(map[int64]bool)
	for _, e := range entries {
		inRound[e.ID] = true
	}

	// Filter to books NOT yet in this round
	var available []dbgen.Book
	for _, b := range allBooks {
		if !inRound[b.ID] {
			available = append(available, b)
		}
	}

	s.renderTemplate(w, "round_books", map[string]interface{}{
		"AdminToken": s.adminToken,
		"Round":      round,
		"Entries":    entries,
		"Available":  available,
		"EntryCount": len(entries),
		"LegacyMode": len(entries) == 0,
	})
}

// handleAdminAddBookToRound adds a library book to a round.
func (s *Server) handleAdminAddBookToRound(w http.ResponseWriter, r *http.Request, roundIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	bookID, err := strconv.ParseInt(r.FormValue("book_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	_, err = s.queries.AddBookToRound(ctx, dbgen.AddBookToRoundParams{
		RoundID: roundID,
		BookID:  bookID,
		AddedBy: toNullString("admin"),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add book: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/round/%d/books", s.adminToken, roundID), http.StatusSeeOther)
}

// handleAdminRemoveBookFromRound removes a book from a round's pool.
func (s *Server) handleAdminRemoveBookFromRound(w http.ResponseWriter, r *http.Request, roundIDStr, bookIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.RemoveBookFromRound(ctx, dbgen.RemoveBookFromRoundParams{
		RoundID: roundID,
		BookID:  bookID,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove book: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/round/%d/books", s.adminToken, roundID), http.StatusSeeOther)
}
