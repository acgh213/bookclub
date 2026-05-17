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

// handleAdminReadingWeeks manages the reading week schedule for a schedule entry.
func (s *Server) handleAdminReadingWeeks(w http.ResponseWriter, r *http.Request, scheduleIDStr string) {
	ctx := r.Context()
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	if r.Method == "POST" {
		weekNumStr := r.FormValue("week_number")
		weekNum, err := strconv.ParseInt(weekNumStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid week number", http.StatusBadRequest)
			return
		}

		_, err = s.queries.CreateReadingWeek(ctx, dbgen.CreateReadingWeekParams{
			ScheduleID:     scheduleID,
			WeekNumber:     weekNum,
			WeekLabel:      toNullString(r.FormValue("week_label")),
			StartChapter:   toNullString(r.FormValue("start_chapter")),
			EndChapter:     toNullString(r.FormValue("end_chapter")),
			Notes:          toNullString(r.FormValue("notes")),
			DiscussionDate: toNullString(r.FormValue("discussion_date")),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create week: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleAdminUpdateWeek updates a reading week entry.
func (s *Server) handleAdminUpdateWeek(w http.ResponseWriter, r *http.Request, weekIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	weekID, err := strconv.ParseInt(weekIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid week ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.UpdateReadingWeek(ctx, dbgen.UpdateReadingWeekParams{
		WeekLabel:      toNullString(r.FormValue("week_label")),
		StartChapter:   toNullString(r.FormValue("start_chapter")),
		EndChapter:     toNullString(r.FormValue("end_chapter")),
		Notes:          toNullString(r.FormValue("notes")),
		DiscussionDate: toNullString(r.FormValue("discussion_date")),
		ID:             weekID,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update week: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

// handleAdminDeleteWeek deletes a reading week entry.
func (s *Server) handleAdminDeleteWeek(w http.ResponseWriter, r *http.Request, weekIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	weekID, err := strconv.ParseInt(weekIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid week ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteReadingWeek(ctx, weekID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete week: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

// handleAdminUpdateScheduleProgress updates total_chapters and cover_url on a schedule entry.
func (s *Server) handleAdminUpdateScheduleProgress(w http.ResponseWriter, r *http.Request, scheduleIDStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	scheduleID, err := strconv.ParseInt(scheduleIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	// Get current entry to preserve other fields
	entry, err := s.queries.GetScheduleEntry(ctx, scheduleID)
	if err != nil {
		http.Error(w, "Schedule entry not found", http.StatusNotFound)
		return
	}

	var totalChapters sql.NullInt64
	if tc := r.FormValue("total_chapters"); tc != "" {
		if n, err := strconv.ParseInt(tc, 10, 64); err == nil {
			totalChapters = sql.NullInt64{Int64: n, Valid: true}
		}
	}

	// Update via raw SQL since the generated UpdateScheduleEntry doesn't include new columns
	_, err = s.db.ExecContext(ctx,
		"UPDATE schedule SET total_chapters = ?, cover_url = ? WHERE id = ?",
		totalChapters, toNullString(r.FormValue("cover_url")), entry.ID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

// handleCheckin handles public check-in POST from the schedule page.
func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	scheduleID, err := strconv.ParseInt(r.FormValue("schedule_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	weekNumber, err := strconv.ParseInt(r.FormValue("week_number"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid week number", http.StatusBadRequest)
		return
	}

	nickname := strings.TrimSpace(r.FormValue("nickname"))
	if nickname == "" {
		http.Error(w, "Nickname required", http.StatusBadRequest)
		return
	}

	emoji := r.FormValue("emoji")
	if emoji == "" {
		emoji = "🦦"
	}

	if err := s.queries.UpsertCheckin(ctx, dbgen.UpsertCheckinParams{
		ScheduleID: scheduleID,
		Nickname:   nickname,
		WeekNumber: weekNumber,
		Emoji:      emoji,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to check in: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON for AJAX or redirect for form submit
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}
