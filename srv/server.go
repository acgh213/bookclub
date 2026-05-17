package srv

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/acgh213/bookclub/db"
	"github.com/acgh213/bookclub/db/dbgen"
)

//go:embed templates/* static/*
var embedFS embed.FS

type Server struct {
	db         *sql.DB
	queries    *dbgen.Queries
	hostname   string
	adminToken string
	templates  map[string]*template.Template
	metadata   BookMetadataService
}

func New(dbPath, hostname string) (*Server, error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.RunMigrations(database); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	s := &Server{
		db:       database,
		queries:  dbgen.New(database),
		hostname: hostname,
		metadata: resolveMetadataService(),
	}

	if err := s.initAdminToken(); err != nil {
		return nil, err
	}

	if err := s.loadTemplates(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Serve(addr string) error {
	log.Printf("\nThe Otter Hole Book Club is running")
	log.Printf("Listening on %s", addr)
	log.Printf("Admin URL: https://%s/admin/%s\n", s.hostname, s.adminToken)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) initAdminToken() error {
	ctx := context.Background()

	// Check env var first
	if token := os.Getenv("ADMIN_TOKEN"); token != "" {
		s.adminToken = token
		log.Printf("Using ADMIN_TOKEN from environment")
		return nil
	}

	// Check database
	cfg, err := s.queries.GetConfig(ctx, "admin_token")
	if err == nil {
		s.adminToken = cfg.Value
		log.Printf("Loaded admin token from database")
		return nil
	}

	// Generate new token
	token, err := generateToken(32)
	if err != nil {
		return err
	}

	if err := s.queries.SetConfig(ctx, dbgen.SetConfigParams{
		Key:   "admin_token",
		Value: token,
	}); err != nil {
		return err
	}

	s.adminToken = token
	log.Printf("\nNew admin token generated")
	log.Printf("Admin URL: https://%s/admin/%s", s.hostname, token)
	log.Printf("Save this token securely.")

	return nil
}

func (s *Server) loadTemplates() error {
	s.templates = make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"jsonStr": func(s string) template.JS {
			b, _ := json.Marshal(s)
			return template.JS(b)
		},
	}

	pages := []string{"home", "submit", "vote", "results", "schedule", "admin", "books", "library", "book_edit", "round_books", "present"}
	for _, page := range pages {
		tmpl, err := template.New("").Funcs(funcMap).ParseFS(embedFS,
			"templates/layout.html",
			fmt.Sprintf("templates/%s.html", page),
		)
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", page, err)
		}
		s.templates[page] = tmpl
	}

	return nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(embedFS)))

	// Public routes
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/submit", s.handleSubmit)
	mux.HandleFunc("/books", s.handleBooks)
	mux.HandleFunc("/vote/", s.handleVote)
	mux.HandleFunc("/results/", s.handleResults)
	mux.HandleFunc("/schedule", s.handleSchedule)

// Public API
	mux.HandleFunc("/checkin", s.handleCheckin)

// API routes
	mux.HandleFunc("/api/round/", s.handleWheelAPI)
	mux.HandleFunc("/api/result/", s.handleResultAPI)

	// Admin routes
	mux.HandleFunc("/admin/", s.handleAdmin)

	return mux
}

// ---------------------------------------------------------------------------
// Public handlers
// ---------------------------------------------------------------------------

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()

	var (
		currentRound    *dbgen.Round
		currentReading  *dbgen.Schedule
		nextBook        *dbgen.Schedule
		currentBook     *dbgen.Book
		submissionCount int64
		voteCount       int64
	)

	if cr, err := s.queries.GetCurrentRound(ctx); err == nil {
		currentRound = &cr
		submissionCount, _ = s.queries.CountSubmissionsByRound(ctx, cr.ID)
		voteCount, _ = s.queries.CountVotesByRound(ctx, cr.ID)
	}
	if sch, err := s.queries.GetCurrentReading(ctx); err == nil {
		currentReading = &sch
	}
	if sch, err := s.queries.GetNextUpcoming(ctx); err == nil {
		nextBook = &sch
	}
	if bk, err := s.queries.GetCurrentBook(ctx); err == nil {
		currentBook = &bk
	}

	s.renderTemplate(w, "home", map[string]interface{}{
		"CurrentRound":    currentRound,
		"CurrentReading":  currentReading,
		"NextBook":        nextBook,
		"CurrentBook":     currentBook,
		"SubmissionCount": submissionCount,
		"VoteCount":       voteCount,
	})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	cr, err := s.queries.GetCurrentRound(ctx)
	if err != nil {
		s.renderTemplate(w, "submit", map[string]interface{}{
			"Error": "Submissions are currently closed. Check back later!",
		})
		return
	}
	if cr.Status != "submissions_open" {
		s.renderTemplate(w, "submit", map[string]interface{}{
			"Error": "Submissions are currently closed. Check back later!",
			"Round": cr,
		})
		return
	}

	if r.Method == "POST" {
		nickname := strings.TrimSpace(r.FormValue("nickname"))
		bookTitle := strings.TrimSpace(r.FormValue("book_title"))
		bookAuthor := strings.TrimSpace(r.FormValue("book_author"))

		if nickname == "" || bookTitle == "" || bookAuthor == "" {
			s.renderTemplate(w, "submit", map[string]interface{}{
				"Error":      "All fields are required.",
				"Round":      cr,
				"Nickname":   nickname,
				"BookTitle":  bookTitle,
				"BookAuthor": bookAuthor,
			})
			return
		}

		_, err := s.queries.CreateSubmission(ctx, dbgen.CreateSubmissionParams{
			RoundID:    cr.ID,
			Nickname:   nickname,
			BookTitle:  bookTitle,
			BookAuthor: bookAuthor,
		})
		if err != nil {
			s.renderTemplate(w, "submit", map[string]interface{}{
				"Error": "Failed to submit book. Please try again.",
				"Round": cr,
			})
			return
		}

		s.renderTemplate(w, "submit", map[string]interface{}{
			"Success": "Book submitted successfully!",
			"Round":   cr,
		})
		return
	}

	s.renderTemplate(w, "submit", map[string]interface{}{
		"Round": cr,
	})
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := strings.TrimPrefix(r.URL.Path, "/vote/")
	if code == "" {
		http.Error(w, "Invalid vote code", http.StatusNotFound)
		return
	}

	round, err := s.queries.GetRoundByVoteCode(ctx, code)
	if err != nil {
		http.Error(w, "Invalid vote code", http.StatusNotFound)
		return
	}

	if round.Status != "voting_open" {
		s.renderTemplate(w, "vote", map[string]interface{}{
			"Error": "Voting is currently closed for this round.",
			"Round": round,
		})
		return
	}

	submissions, err := s.queries.ListSubmissionsByRound(ctx, round.ID)
	if err != nil || len(submissions) == 0 {
		s.renderTemplate(w, "vote", map[string]interface{}{
			"Error": "No books have been submitted for this round yet.",
			"Round": round,
		})
		return
	}

	if r.Method == "POST" {
		nickname := strings.TrimSpace(r.FormValue("nickname"))
		if nickname == "" {
			s.renderTemplate(w, "vote", map[string]interface{}{
				"Error":       "Nickname is required.",
				"Round":       round,
				"Submissions": submissions,
			})
			return
		}

		// Duplicate vote check
		_, dupErr := s.queries.GetVoteByNickname(ctx, dbgen.GetVoteByNicknameParams{
			RoundID:  round.ID,
			Nickname: nickname,
		})
		if dupErr == nil {
			s.renderTemplate(w, "vote", map[string]interface{}{
				"Error":       "You've already voted in this round.",
				"Round":       round,
				"Submissions": submissions,
			})
			return
		}

		// Parse rankings
		numBooks := int64(len(submissions))
		rankings := make(map[int64]int64) // submissionID -> rank
		for _, sub := range submissions {
			rankStr := r.FormValue(fmt.Sprintf("rank_%d", sub.ID))
			rank, err := strconv.ParseInt(rankStr, 10, 64)
			if err != nil || rank < 1 || rank > numBooks {
				s.renderTemplate(w, "vote", map[string]interface{}{
					"Error":       fmt.Sprintf("Please rank all books from 1 to %d!.", numBooks),
					"Round":       round,
					"Submissions": submissions,
				})
				return
			}
			rankings[sub.ID] = rank
		}

		// Verify all ranks are unique
		usedRanks := make(map[int64]bool)
		for _, rank := range rankings {
			if usedRanks[rank] {
				s.renderTemplate(w, "vote", map[string]interface{}{
					"Error":       "Each rank must be used exactly once.",
					"Round":       round,
					"Submissions": submissions,
				})
				return
			}
			usedRanks[rank] = true
		}

		// Create vote record
		vote, err := s.queries.CreateVote(ctx, dbgen.CreateVoteParams{
			RoundID:  round.ID,
			Nickname: nickname,
		})
		if err != nil {
			s.renderTemplate(w, "vote", map[string]interface{}{
				"Error":       "Failed to save vote. Please try again.",
				"Round":       round,
				"Submissions": submissions,
			})
			return
		}

		// Save each ranking
		for subID, rank := range rankings {
			if err := s.queries.CreateVoteRanking(ctx, dbgen.CreateVoteRankingParams{
				VoteID:       vote.ID,
				SubmissionID: subID,
				Rank:         rank,
			}); err != nil {
				log.Printf("failed to save ranking: %v", err)
				http.Error(w, "Failed to save rankings", http.StatusInternalServerError)
				return
			}
		}

		s.renderTemplate(w, "vote", map[string]interface{}{
			"Success":     "Vote submitted successfully!",
			"Round":       round,
			"Submissions": submissions,
		})
		return
	}

	s.renderTemplate(w, "vote", map[string]interface{}{
		"Round":       round,
		"Submissions": submissions,
	})
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := strings.TrimPrefix(r.URL.Path, "/results/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusNotFound)
		return
	}

	round, err := s.queries.GetRound(ctx, id)
	if err != nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	submissions, err := s.queries.ListSubmissionsByRound(ctx, round.ID)
	if err != nil {
		http.Error(w, "Failed to load submissions", http.StatusInternalServerError)
		return
	}

	votes, err := s.queries.ListVotesByRound(ctx, round.ID)
	if err != nil {
		http.Error(w, "Failed to load votes", http.StatusInternalServerError)
		return
	}

	allRankings, err := s.queries.ListAllRankingsByRound(ctx, round.ID)
	if err != nil {
		http.Error(w, "Failed to load rankings", http.StatusInternalServerError)
		return
	}

	// Group rankings by vote
	voteRankingsMap := make(map[int64][]dbgen.VoteRanking)
	for _, ranking := range allRankings {
		voteRankingsMap[ranking.VoteID] = append(voteRankingsMap[ranking.VoteID], ranking)
	}

	// Build ballots
	var ballots []Ballot
	for _, v := range votes {
		ranks := voteRankingsMap[v.ID]
		sort.Slice(ranks, func(i, j int) bool {
			return ranks[i].Rank < ranks[j].Rank
		})
		var choices []int64
		for _, rk := range ranks {
			choices = append(choices, rk.SubmissionID)
		}
		ballots = append(ballots, Ballot{Choices: choices})
	}

	// Submission lookup map
	subMap := make(map[int64]dbgen.Submission)
	for _, sub := range submissions {
		subMap[sub.ID] = sub
	}

	result := runRankedChoiceVoting(ballots, subMap)

	s.renderTemplate(w, "results", map[string]interface{}{
		"Round":       round,
		"Submissions": submissions,
		"VoteCount":   len(votes),
		"Result":      result,
	})
}

func (s *Server) handleBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type RoundGroup struct {
		RoundTitle  string
		RoundStatus string
		Books       []dbgen.ListAllSubmissionsRow
	}

	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "recent"
	}

	var libraryBooks []dbgen.Book
	var hasLibrary bool

	if query != "" {
		q := sql.NullString{String: query, Valid: true}
		results, err := s.queries.SearchBooks(ctx, dbgen.SearchBooksParams{
			Column1: q,
			Column2: q,
			Column3: q,
		})
		if err == nil {
			libraryBooks = results
			hasLibrary = len(libraryBooks) > 0
		}
	} else if status != "" {
		results, err := s.queries.ListBooksByStatus(ctx, status)
		if err == nil {
			libraryBooks = results
			hasLibrary = len(libraryBooks) > 0
		}
	} else {
		results, err := s.queries.ListBooks(ctx)
		if err == nil {
			libraryBooks = results
			hasLibrary = len(libraryBooks) > 0
		}
	}

	// Sort library books
	if sortBy == "title" {
		sort.Slice(libraryBooks, func(i, j int) bool {
			return strings.ToLower(libraryBooks[i].Title) < strings.ToLower(libraryBooks[j].Title)
		})
	} else if sortBy == "author" {
		sort.Slice(libraryBooks, func(i, j int) bool {
			return strings.ToLower(libraryBooks[i].Author) < strings.ToLower(libraryBooks[j].Author)
		})
	}

	// Legacy submissions (shown below library if no library books)
	var groups []RoundGroup
	if !hasLibrary {
		legacyBooks, _ := s.queries.ListAllSubmissions(ctx)
		groupMap := make(map[string]int)
		for _, b := range legacyBooks {
			idx, ok := groupMap[b.RoundTitle]
			if !ok {
				idx = len(groups)
				groupMap[b.RoundTitle] = idx
				groups = append(groups, RoundGroup{
					RoundTitle:  b.RoundTitle,
					RoundStatus: b.RoundStatus,
				})
			}
			groups[idx].Books = append(groups[idx].Books, b)
		}
	}

	s.renderTemplate(w, "books", map[string]interface{}{
		"Groups":       groups,
		"HasBooks":     hasLibrary || len(groups) > 0,
		"LibraryBooks": libraryBooks,
		"HasLibrary":   hasLibrary,
		"Query":        query,
		"Status":       status,
		"Sort":         sortBy,
		"BookCount":    len(libraryBooks),
	})
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	schedule, err := s.queries.ListSchedule(ctx)
	if err != nil {
		http.Error(w, "Failed to load schedule", http.StatusInternalServerError)
		return
	}

	// Build enriched schedule data with weeks and check-ins
	type WeekWithCheckins struct {
		Week     dbgen.ReadingWeek
		Checkins []dbgen.Checkin
	}
	type ScheduleEntry struct {
		Schedule      dbgen.Schedule
		Weeks         []WeekWithCheckins
		CurrentWeek   *dbgen.ReadingWeek
		CurrentWeekNum int64
		TotalWeeks    int
		ProgressPct   int
	}

	var entries []ScheduleEntry
	for _, sch := range schedule {
		entry := ScheduleEntry{Schedule: sch}
		weeks, _ := s.queries.ListReadingWeeks(ctx, sch.ID)
		for _, w := range weeks {
			checkins, _ := s.queries.ListCheckinsByWeek(ctx, sch.ID, w.WeekNumber)
			entry.Weeks = append(entry.Weeks, WeekWithCheckins{
				Week:     w,
				Checkins: checkins,
			})
		}
		entry.TotalWeeks = len(weeks)
		// Find the "current" week: the earliest week that has checkins, or week 1
		// For progress, find highest week with any checkins
		currentWeekNum := int64(1)
		for _, wc := range entry.Weeks {
			if len(wc.Checkins) > 0 {
				currentWeekNum = wc.Week.WeekNumber
			}
		}
		entry.CurrentWeekNum = currentWeekNum
		// Set current week to the one we're on
		for i := range weeks {
			if weeks[i].WeekNumber == currentWeekNum {
				entry.CurrentWeek = &weeks[i]
				break
			}
		}
		// Calculate progress: use current week's end chapter
		if entry.CurrentWeek != nil && sch.TotalChapters.Valid && sch.TotalChapters.Int64 > 0 && entry.CurrentWeek.EndChapter.Valid {
			entry.ProgressPct = calcProgressPct(entry.CurrentWeek.EndChapter.String, sch.TotalChapters.Int64)
		}
		entries = append(entries, entry)
	}

	s.renderTemplate(w, "schedule", map[string]interface{}{
		"Entries": entries,
	})
}

// ---------------------------------------------------------------------------
// Admin handlers
// ---------------------------------------------------------------------------

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] != s.adminToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if len(parts) == 1 {
		s.handleAdminDashboard(w, r)
		return
	}

	switch parts[1] {
	case "round":
		if len(parts) == 2 {
			s.handleAdminCreateRound(w, r)
		} else if len(parts) == 4 && parts[3] == "status" {
			s.handleAdminUpdateRoundStatus(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "books" {
			s.handleAdminRoundBooks(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "present" {
			s.handleAdminPresent(w, r, parts[2])
		} else if len(parts) == 5 && parts[3] == "books" && parts[4] == "add" {
			s.handleAdminAddBookToRound(w, r, parts[2])
		} else if len(parts) == 6 && parts[3] == "books" && parts[5] == "remove" {
			s.handleAdminRemoveBookFromRound(w, r, parts[2], parts[4])
		} else {
			http.NotFound(w, r)
		}
	case "book":
		if len(parts) == 2 && r.Method == "GET" {
			s.handleAdminLookupBook(w, r)
		} else if len(parts) == 2 && r.Method == "POST" {
			// POST is only for lookup with q param — handled above
			s.handleAdminLookupBook(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "books":
		if len(parts) == 2 {
			s.handleAdminBooks(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "library":
		if len(parts) == 2 {
			s.handleAdminCreateBook(w, r)
		} else if len(parts) == 3 {
			s.handleAdminEditBook(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "archive" {
			s.handleAdminArchiveBook(w, r, parts[2])
		} else {
			http.NotFound(w, r)
		}
	case "import":
		if len(parts) == 2 && r.Method == "POST" {
			s.handleAdminImportBook(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "submission":
		if len(parts) == 3 {
			s.handleAdminUpdateSubmission(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "delete" {
			s.handleAdminDeleteSubmission(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "promote" {
			s.handleAdminPromoteSubmission(w, r, parts[2])
		} else {
			http.NotFound(w, r)
		}
	case "schedule":
		if len(parts) == 2 {
			s.handleAdminCreateSchedule(w, r)
		} else if len(parts) == 3 {
			s.handleAdminUpdateSchedule(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "delete" {
			s.handleAdminDeleteSchedule(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "weeks" {
			s.handleAdminReadingWeeks(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "progress" {
			s.handleAdminUpdateScheduleProgress(w, r, parts[2])
		} else {
			http.NotFound(w, r)
		}
	case "week":
		if len(parts) == 3 {
			s.handleAdminUpdateWeek(w, r, parts[2])
		} else if len(parts) == 4 && parts[3] == "delete" {
			s.handleAdminDeleteWeek(w, r, parts[2])
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rounds, _ := s.queries.ListRounds(ctx)
	schedule, _ := s.queries.ListSchedule(ctx)

	roundData := make([]map[string]interface{}, len(rounds))
	for i, round := range rounds {
		subCount, _ := s.queries.CountSubmissionsByRound(ctx, round.ID)
		voteCount, _ := s.queries.CountVotesByRound(ctx, round.ID)
		submissions, _ := s.queries.ListSubmissionsByRound(ctx, round.ID)
		roundData[i] = map[string]interface{}{
			"Round":           round,
			"SubmissionCount": subCount,
			"VoteCount":       voteCount,
			"Submissions":     submissions,
		}
	}

	// Enrich schedule with reading weeks
	type ScheduleWithWeeks struct {
		Schedule dbgen.Schedule
		Weeks    []dbgen.ReadingWeek
		NextWeek int64
	}
	var enrichedSchedule []ScheduleWithWeeks
	for _, sch := range schedule {
		weeks, _ := s.queries.ListReadingWeeks(ctx, sch.ID)
		nextWeek := int64(1)
		if len(weeks) > 0 {
			nextWeek = weeks[len(weeks)-1].WeekNumber + 1
		}
		enrichedSchedule = append(enrichedSchedule, ScheduleWithWeeks{
			Schedule: sch,
			Weeks:    weeks,
			NextWeek: nextWeek,
		})
	}

	s.renderTemplate(w, "admin", map[string]interface{}{
		"AdminToken": s.adminToken,
		"Hostname":   s.hostname,
		"Rounds":     roundData,
		"Schedule":   enrichedSchedule,
	})
}

func (s *Server) handleAdminCreateRound(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
		return
	}

	voteCode, err := generateToken(8)
	if err != nil {
		http.Error(w, "Failed to generate vote code", http.StatusInternalServerError)
		return
	}

	_, err = s.queries.CreateRound(ctx, dbgen.CreateRoundParams{
		Title:    title,
		Status:   "submissions_open",
		VoteCode: voteCode,
	})
	if err != nil {
		http.Error(w, "Failed to create round", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminUpdateRoundStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid round ID", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status != "submissions_open" && status != "voting_open" && status != "closed" {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	if err := s.queries.UpdateRoundStatus(ctx, dbgen.UpdateRoundStatusParams{
		Status: status,
		ID:     id,
	}); err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminCreateSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	var roundID sql.NullInt64
	if ridStr := r.FormValue("round_id"); ridStr != "" {
		if rid, err := strconv.ParseInt(ridStr, 10, 64); err == nil {
			roundID = sql.NullInt64{Int64: rid, Valid: true}
		}
	}

	suggestedBy := r.FormValue("suggested_by")
	meetingDate := r.FormValue("meeting_date")
	readingProgress := r.FormValue("reading_progress")
	notes := r.FormValue("notes")

	_, err := s.queries.CreateScheduleEntry(ctx, dbgen.CreateScheduleEntryParams{
		RoundID:         roundID,
		BookTitle:       r.FormValue("book_title"),
		BookAuthor:      r.FormValue("book_author"),
		SuggestedBy:     sql.NullString{String: suggestedBy, Valid: suggestedBy != ""},
		MeetingDate:     sql.NullString{String: meetingDate, Valid: meetingDate != ""},
		ReadingProgress: sql.NullString{String: readingProgress, Valid: readingProgress != ""},
		Status:          r.FormValue("status"),
		Notes:           sql.NullString{String: notes, Valid: notes != ""},
	})
	if err != nil {
		http.Error(w, "Failed to create schedule entry", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminUpdateSchedule(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	suggestedBy := r.FormValue("suggested_by")
	meetingDate := r.FormValue("meeting_date")
	readingProgress := r.FormValue("reading_progress")
	notes := r.FormValue("notes")

	if err := s.queries.UpdateScheduleEntry(ctx, dbgen.UpdateScheduleEntryParams{
		BookTitle:       r.FormValue("book_title"),
		BookAuthor:      r.FormValue("book_author"),
		SuggestedBy:     sql.NullString{String: suggestedBy, Valid: suggestedBy != ""},
		MeetingDate:     sql.NullString{String: meetingDate, Valid: meetingDate != ""},
		ReadingProgress: sql.NullString{String: readingProgress, Valid: readingProgress != ""},
		Status:          r.FormValue("status"),
		Notes:           sql.NullString{String: notes, Valid: notes != ""},
		ID:              id,
	}); err != nil {
		http.Error(w, "Failed to update schedule entry", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminUpdateSubmission(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.UpdateSubmission(ctx, dbgen.UpdateSubmissionParams{
		BookTitle:  r.FormValue("book_title"),
		BookAuthor: r.FormValue("book_author"),
		Nickname:   r.FormValue("nickname"),
		ID:         id,
	}); err != nil {
		http.Error(w, "Failed to update submission", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminDeleteSubmission(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid submission ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteSubmission(ctx, id); err != nil {
		http.Error(w, "Failed to delete submission", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

func (s *Server) handleAdminDeleteSchedule(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteScheduleEntry(ctx, id); err != nil {
		http.Error(w, "Failed to delete schedule entry", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/%s", s.adminToken), http.StatusSeeOther)
}

// ---------------------------------------------------------------------------
// Template rendering
// ---------------------------------------------------------------------------

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	tmpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Template error (%s): %v", name, err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// calcProgressPct extracts a number from a string like "Chapter 12" or "12" and returns a percentage.
func calcProgressPct(endChapter string, total int64) int {
	// Try bare number first
	if n, err := strconv.ParseInt(strings.TrimSpace(endChapter), 10, 64); err == nil {
		pct := int(n * 100 / total)
		if pct > 100 {
			return 100
		}
		return pct
	}
	// Try extracting last number from string like "Chapter 12"
	parts := strings.Fields(endChapter)
	for i := len(parts) - 1; i >= 0; i-- {
		if n, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
			pct := int(n * 100 / total)
			if pct > 100 {
				return 100
			}
			return pct
		}
	}
	return 0
}

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:length], nil
}

// ---------------------------------------------------------------------------
// Ranked Choice Voting
// ---------------------------------------------------------------------------

// Ballot holds one voter's ordered choices (first element = top pick).
type Ballot struct {
	Choices []int64
}

// EliminationRound captures what happened in one round of counting.
type EliminationRound struct {
	RoundNumber int
	Counts      map[int64]int
	Eliminated  int64 // 0 if no elimination (winner found)
	Winner      int64 // 0 if no winner yet
}

// VotingResult is the full output of the ranked-choice algorithm.
type VotingResult struct {
	Rounds []EliminationRound
	Winner int64
}

func runRankedChoiceVoting(ballots []Ballot, submissions map[int64]dbgen.Submission) VotingResult {
	if len(ballots) == 0 {
		return VotingResult{}
	}

	active := make(map[int64]bool)
	for id := range submissions {
		active[id] = true
	}

	var rounds []EliminationRound
	roundNum := 1

	for len(active) > 1 {
		// Tally first-choice votes among active candidates.
		counts := make(map[int64]int)
		for id := range active {
			counts[id] = 0
		}
		for _, ballot := range ballots {
			for _, choice := range ballot.Choices {
				if active[choice] {
					counts[choice]++
					break
				}
			}
		}

		totalVotes := len(ballots)

		// Check for majority winner (>50%).
		for id, count := range counts {
			if count*2 > totalVotes {
				rounds = append(rounds, EliminationRound{
					RoundNumber: roundNum,
					Counts:      counts,
					Winner:      id,
				})
				return VotingResult{Rounds: rounds, Winner: id}
			}
		}

		// Eliminate the candidate with fewest first-choice votes.
		var minID int64
		minCount := totalVotes + 1
		for id, count := range counts {
			if count < minCount {
				minCount = count
				minID = id
			}
		}

		rounds = append(rounds, EliminationRound{
			RoundNumber: roundNum,
			Counts:      counts,
			Eliminated:  minID,
		})

		delete(active, minID)
		roundNum++
	}

	// Last one standing.
	var winnerID int64
	for id := range active {
		winnerID = id
	}

	// Final tally round.
	finalCounts := make(map[int64]int)
	for _, ballot := range ballots {
		for _, choice := range ballot.Choices {
			if active[choice] {
				finalCounts[choice]++
				break
			}
		}
	}

	rounds = append(rounds, EliminationRound{
		RoundNumber: roundNum,
		Counts:      finalCounts,
		Winner:      winnerID,
	})

	return VotingResult{Rounds: rounds, Winner: winnerID}
}
