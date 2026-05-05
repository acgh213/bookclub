package srv

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/acgh213/bookclub/db/dbgen"
)

// --- Admin book library handlers ---

func (s *Server) handleAdminBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	books, err := s.queries.ListBooks(ctx)
	if err != nil {
		http.Error(w, "Failed to load library", http.StatusInternalServerError)
		return
	}

	s.renderTemplate(w, "library", map[string]interface{}{
		"AdminToken": s.adminToken,
		"Books":      books,
		"BookCount":  len(books),
	})
}

func (s *Server) handleAdminCreateBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	title := strings.TrimSpace(r.FormValue("title"))
	author := strings.TrimSpace(r.FormValue("author"))
	if title == "" || author == "" {
		http.Error(w, "Title and author are required", http.StatusBadRequest)
		return
	}

	tags := r.FormValue("tags")
	pageCountStr := r.FormValue("page_count")
	var pageCount sql.NullInt64
	if pageCountStr != "" {
		if n, err := strconv.ParseInt(pageCountStr, 10, 64); err == nil {
			pageCount = sql.NullInt64{Int64: n, Valid: true}
		}
	}

	_, err := s.queries.CreateBook(ctx, dbgen.CreateBookParams{
		Title:          title,
		Author:         author,
		Submitter:       toNullString(r.FormValue("submitter")),
		Pitch:          toNullString(r.FormValue("pitch")),
		Tags:           toNullString(tags),
		PageCount:      pageCount,
		Description:    toNullString(r.FormValue("description")),
		ContentNotes:   toNullString(r.FormValue("content_notes")),
		CoverUrl:       toNullString(r.FormValue("cover_url")),
		Isbn:           toNullString(r.FormValue("isbn")),
		MetadataSource: toNullString(r.FormValue("metadata_source")),
		MetadataID:     toNullString(r.FormValue("metadata_id")),
		Status:         "proposed",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create book: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/books", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminEditBook(w http.ResponseWriter, r *http.Request, idStr string) {
	ctx := r.Context()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if r.Method == "POST" {
		title := strings.TrimSpace(r.FormValue("title"))
		author := strings.TrimSpace(r.FormValue("author"))
		if title == "" || author == "" {
			http.Error(w, "Title and author are required", http.StatusBadRequest)
			return
		}

		pageCountStr := r.FormValue("page_count")
		var pageCount sql.NullInt64
		if pageCountStr != "" {
			if n, err := strconv.ParseInt(pageCountStr, 10, 64); err == nil {
				pageCount = sql.NullInt64{Int64: n, Valid: true}
			}
		}

		if err := s.queries.UpdateBook(ctx, dbgen.UpdateBookParams{
			Title:          title,
			Author:         author,
			Submitter:       toNullString(r.FormValue("submitter")),
			Pitch:          toNullString(r.FormValue("pitch")),
			Tags:           toNullString(r.FormValue("tags")),
			PageCount:      pageCount,
			Description:    toNullString(r.FormValue("description")),
			ContentNotes:   toNullString(r.FormValue("content_notes")),
			CoverUrl:       toNullString(r.FormValue("cover_url")),
			Isbn:           toNullString(r.FormValue("isbn")),
			MetadataSource: toNullString(r.FormValue("metadata_source")),
			MetadataID:     toNullString(r.FormValue("metadata_id")),
			Status:         r.FormValue("status"),
			ID:             id,
		}); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update book: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/admin/%s/books", s.adminToken), http.StatusSeeOther)
		return
	}

	// GET — show edit form
	book, err := s.queries.GetBook(ctx, id)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	s.renderTemplate(w, "book_edit", map[string]interface{}{
		"AdminToken": s.adminToken,
		"Book":       book,
	})
}

func (s *Server) handleAdminArchiveBook(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid book ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.ArchiveBook(ctx, id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to archive book: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/books", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminImportBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	title := strings.TrimSpace(r.FormValue("title"))
	author := strings.TrimSpace(r.FormValue("author"))
	if title == "" || author == "" {
		http.Error(w, "Title and author are required", http.StatusBadRequest)
		return
	}

	pageCountStr := r.FormValue("page_count")
	var pageCount sql.NullInt64
	if pageCountStr != "" {
		if n, err := strconv.ParseInt(pageCountStr, 10, 64); err == nil {
			pageCount = sql.NullInt64{Int64: n, Valid: true}
		}
	}

	_, err := s.queries.CreateBook(ctx, dbgen.CreateBookParams{
		Title:          title,
		Author:         author,
		Description:    toNullString(r.FormValue("description")),
		CoverUrl:       toNullString(r.FormValue("cover_url")),
		PageCount:      pageCount,
		Isbn:           toNullString(r.FormValue("isbn")),
		MetadataSource: toNullString(r.FormValue("metadata_source")),
		MetadataID:     toNullString(r.FormValue("metadata_id")),
		Status:         "proposed",
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to import book: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s/books", s.adminToken), http.StatusSeeOther)
}

// toNullString returns sql.NullString from a string value.
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
