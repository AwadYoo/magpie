package sessions

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Kimi Code's sessions are listed as the calls' folders name them (#1913,
// infinitr0us): the state.json holds the id and the times (1.5) or the
// folder names them (1.4), the title the agent made, the cwd, and the
// tokens come from the wire files' usage records — the main file's and the
// subagents'.
func TestKimiCodeSessions(t *testing.T) {
	kimiHome(t)
	Reset()
	ss := List(0)
	if n := count(ss, "kimi"); n != 8 {
		t.Fatalf("kimi sessions %d, want 8 (the six with turns, the no-turn one and the migrated one): %+v", n, ss)
	}
	const june = "session_a0a7c8d3-2a5a-480d-9d42-740257598abd"
	s, ok := findSession(ss, "kimi", june)
	if !ok {
		t.Fatalf("june's session isn't listed; have %+v", ss)
	}
	// the call reader's tokens for the same session (calls_kimi_test.go's
	// totals), its subagent's included
	if want := (kimiWant[june]); s.Tokens != (Tokens{Input: want.input, CacheRead: want.cacheRead, CacheWrite: want.cacheWrite, Output: want.output}) {
		t.Fatalf("tokens %+v, want %+v", s.Tokens, kimiWant[june])
	}
	// 1.4's states keep no cwd (its workDir redacted here); 1.5's do
	v15, ok := findSession(ss, "kimi", "session_fc16731e-2bfe-4873-a6cb-4b9e73280360")
	if !ok || v15.Cwd == "" {
		t.Fatalf("the 1.5 session's cwd: %+v", v15)
	}
	if s.Start.IsZero() || s.Last.IsZero() {
		t.Fatalf("times: %v %v", s.Start, s.Last)
	}
	if s.Resume == "" || !strings.Contains(s.Resume, "kimi --session "+june) {
		t.Fatalf("resume %q", s.Resume)
	}
}

// A session the state doesn't name (1.4) is known by its folder.
func TestKimiSessionNamedByItsFolder(t *testing.T) {
	kimiHome(t)
	Reset()
	ss := List(0)
	for _, s := range ss {
		if s.Agent != "kimi" {
			continue
		}
		if filepath.Base(filepath.Dir(filepath.Dir(s.Path))) == "" {
			t.Fatalf("no folder: %+v", s)
		}
	}
}

// A call made through magpie's entry is the gateway's to count: its model
// names magpie's, and this reader leaves it alone, as the call reader does.
func TestKimiSessionSkipsACallThroughMagpie(t *testing.T) {
	kimiHome(t)
	Reset()
	ss := List(0)
	s, ok := findSession(ss, "kimi", "session_7d423a2a-9c30-4478-8fb0-ceb78f9dbb6a")
	if !ok {
		t.Fatalf("the no-turn session isn't listed: %+v", ss)
	}
	if !s.Tokens.zero() {
		t.Fatalf("tokens counted before any record: %+v", s.Tokens)
	}
	wire := filepath.Join(filepath.Dir(s.Path), "agents", "main", "wire.jsonl")
	b, err := os.ReadFile(wire)
	if err != nil {
		t.Fatal(err)
	}
	rec := `{"type":"usage.record","agentId":"main","model":"magpie/deepseek/flash","usage":{"inputOther":100,"output":10,"inputCacheRead":0,"inputCacheCreation":0},"time":1790830444000}`
	if err := os.WriteFile(wire, append(bytes.TrimRight(b, "\n"), []byte("\n"+rec+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	Reset()
	s, ok = findSession(List(0), "kimi", "session_7d423a2a-9c30-4478-8fb0-ceb78f9dbb6a")
	if !ok || !s.Tokens.zero() {
		t.Fatalf("a magpie call was counted: %+v", s.Tokens)
	}
}
