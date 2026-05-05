package srv

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/acgh213/bookclub/db/dbgen"
)

// handleAdminPromoteSubmission promotes a submission to a library book.
func (s *Server) handleAdminPromoteSubmission(w http.ResponseWriter, r *http.Request, subIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	subID, err := strconv.ParseInt(subIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	// Get the submission
	sub, err := s.queries.GetSubmission(ctx, subID)
	if err != nil {
		http.Error(w, "Submission not found", http.StatusNotFound)
		return
	}

	// Check if already promoted
	_, err = s.queries.GetSubmissionByNickname(ctx, dbgen.GetSubmissionByNicknameParams{
		RoundID:  sub.RoundID,
		Nickname: sub.Nickname + "_promoted",
	})
	if err == nil {
		// Entry exists — book already promoted
		http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
		return
	}

	// Create book from submission
	_, err = s.queries.CreateBook(ctx, dbgen.CreateBookParams{
		Title:          sub.BookTitle,
		Author:         sub.BookAuthor,
		Submitter:       toNullString(sub.Nickname),
		MetadataSource: toNullString("submission"),
		Status:         "proposed",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to promote: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/books", s.adminToken), http.StatusSeeOther)
}
