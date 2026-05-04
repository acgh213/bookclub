package srv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/acgh213/bookclub/db/dbgen"
)

func TestServerSetup(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_server.sqlite3")
	t.Cleanup(func() { os.Remove(tempDB) })

	server, err := New(tempDB, "test-hostname")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.adminToken == "" {
		t.Error("expected admin token to be set")
	}

	t.Run("home page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("submit page no round", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/submit", nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("schedule page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("admin unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/wrong-token", nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("admin authorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/"+server.adminToken, nil)
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestRankedChoiceVoting(t *testing.T) {
	t.Run("majority first round", func(t *testing.T) {
		subs := map[int64]dbgen.Submission{
			1: {ID: 1, BookTitle: "Book A"},
			2: {ID: 2, BookTitle: "Book B"},
		}
		ballots := []Ballot{
			{Choices: []int64{1, 2}},
			{Choices: []int64{1, 2}},
			{Choices: []int64{2, 1}},
		}
		result := runRankedChoiceVoting(ballots, subs)

		if result.Winner != 1 {
			t.Errorf("expected winner 1, got %d", result.Winner)
		}
	})

	t.Run("elimination needed", func(t *testing.T) {
		subs := map[int64]dbgen.Submission{
			1: {ID: 1, BookTitle: "Book A"},
			2: {ID: 2, BookTitle: "Book B"},
			3: {ID: 3, BookTitle: "Book C"},
		}
		ballots := []Ballot{
			{Choices: []int64{1, 2, 3}},
			{Choices: []int64{2, 1, 3}},
			{Choices: []int64{3, 1, 2}},
			{Choices: []int64{1, 3, 2}},
			{Choices: []int64{2, 3, 1}},
		}
		result := runRankedChoiceVoting(ballots, subs)

		if result.Winner == 0 {
			t.Error("expected a winner")
		}
		if len(result.Rounds) < 2 {
			t.Error("expected at least 2 rounds")
		}
	})

	t.Run("no ballots", func(t *testing.T) {
		subs := map[int64]dbgen.Submission{
			1: {ID: 1, BookTitle: "Book A"},
		}
		result := runRankedChoiceVoting(nil, subs)
		if result.Winner != 0 {
			t.Error("expected no winner with no ballots")
		}
	})
}
