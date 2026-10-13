package sessions

import (
	"path/filepath"
	"testing"
)

// MiniMax Code's sessions are listed as its messages files name them
// (#1913): the id is the folder's, the times the messages' timestamps, the
// tokens the assistant messages' usage — a call through magpie's entry is
// the gateway's to count, as the call reader says — and the title the
// first prompt that isn't a system reminder.
func TestMiniMaxCodeSessions(t *testing.T) {
	claude, _ := setup(t)
	dir := filepath.Join(filepath.Dir(claude), "home", ".minimax")
	copyTree(t, "testdata/minimax", dir)
	Reset()
	ss := List(0)
	if n := count(ss, "minimax-code"); n != 1 {
		t.Fatalf("minimax-code sessions %d, want 1 (the one with a call of its own): %+v", n, ss)
	}
	const id = "mvs_82c610ef258f4350b76250bda12cacb5"
	s, ok := findSession(ss, "minimax-code", id)
	if !ok {
		t.Fatalf("the relay session isn't listed; have %+v", ss)
	}
	// the call reader's one call, its tokens here too
	if want := (Tokens{Input: 123, Output: 7, CacheRead: 50, CacheWrite: 20}); s.Tokens != want {
		t.Fatalf("tokens %+v, want %+v", s.Tokens, want)
	}
	if len(s.Models) != 1 || s.Models[0].Model != "m2" {
		t.Fatalf("models %+v", s.Models)
	}
	if s.Start.IsZero() || s.Last.IsZero() {
		t.Fatalf("times: %v %v", s.Start, s.Last)
	}
	// a call through magpie's entry is the gateway's to count: the
	// session that made only those has no tokens and no title here, and
	// is left unlisted
	if _, ok := findSession(ss, "minimax-code", "mvs_a67133b1217e4dafb20dc58758717933"); ok {
		t.Fatalf("the magpie-only session was listed: %+v", ss)
	}
}
