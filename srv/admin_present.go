package srv

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/acgh213/bookclub/db/dbgen"
)

// handleAdminPresent shows the fullscreen presentation mode for a round.
func (s *Server) handleAdminPresent(w http.ResponseWriter, r *http.Request, roundIDStr string) {
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

	// Check for existing preview result
	latest, err := s.queries.GetLatestRoundResult(ctx, roundID)
	var preview *dbgen.RoundResult
	if err == nil && latest.Confirmed == 0 {
		preview = &latest
	}

	// Check for locked winner
	winner, err := s.queries.GetConfirmedResult(ctx, roundID)
	var lockedWinner *dbgen.RoundResult
	if err == nil {
		lockedWinner = &winner
	}

	s.renderTemplate(w, "present", map[string]interface{}{
		"AdminToken": s.adminToken,
		"Round":      round,
		"Preview":    preview,
		"Winner":     lockedWinner,
	})
}

// handleConfirmResult locks a round result (confirmed=1).
func (s *Server) handleConfirmResult(w http.ResponseWriter, r *http.Request, resultIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	resultID, err := strconv.ParseInt(resultIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid result ID", http.StatusBadRequest)
		return
	}

	var body struct {
		ConfirmedBy string `json:"confirmed_by"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.ConfirmedBy == "" {
		body.ConfirmedBy = "admin"
	}

	if err := s.queries.ConfirmResult(ctx, dbgen.ConfirmResultParams{
		ConfirmedBy: sql.NullString{String: body.ConfirmedBy, Valid: true},
		ID:          resultID,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to confirm result: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
}

// handleSetCurrentBook marks the confirmed winner as the current reading.
func (s *Server) handleSetCurrentBook(w http.ResponseWriter, r *http.Request, roundID int64) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Get confirmed winner
	winner, err := s.queries.GetConfirmedResult(ctx, roundID)
	if err != nil {
		http.Error(w, "No confirmed result to set as current book", http.StatusBadRequest)
		return
	}

	// Get the book
	book, err := s.queries.GetBook(ctx, winner.BookID.Int64)
	if err != nil {
		http.Error(w, "Book not found", http.StatusInternalServerError)
		return
	}

	// Mark old current reading as completed
	oldReading, _ := s.queries.GetCurrentReading(ctx)
	if oldReading.ID != 0 {
		s.queries.UpdateScheduleEntry(ctx, dbgen.UpdateScheduleEntryParams{
			BookTitle:       oldReading.BookTitle,
			BookAuthor:      oldReading.BookAuthor,
			SuggestedBy:     oldReading.SuggestedBy,
			MeetingDate:     oldReading.MeetingDate,
			ReadingProgress: oldReading.ReadingProgress,
			Status:          "completed",
			Notes:           oldReading.Notes,
			ID:              oldReading.ID,
		})
	}

	// Create new schedule entry as "reading"
	_, err = s.queries.CreateScheduleEntry(ctx, dbgen.CreateScheduleEntryParams{
		RoundID:         sql.NullInt64{Int64: roundID, Valid: true},
		BookTitle:       book.Title,
		BookAuthor:      book.Author,
		SuggestedBy:     book.Submitter,
		Status:          "reading",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to set current book: %v", err), http.StatusInternalServerError)
		return
	}

	// Update book status to selected
	s.queries.UpdateBookStatus(ctx, dbgen.UpdateBookStatusParams{
		Status: "selected",
		ID:     book.ID,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "set_as_current"})
}

// handleResultAPI dispatches /api/result/{id}/confirm
func (s *Server) handleResultAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/result/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 2 && parts[1] == "confirm" && r.Method == http.MethodPost {
		s.handleConfirmResult(w, r, parts[0])
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
