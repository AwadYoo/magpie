package sessions

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sessions/<workdir slug>/<session id>/ of Kimi's home, whose state.json
// names it: the id, the cwd (workDir before version 2, cwd since), the
// times in epoch milliseconds, the title (the user's when isCustomTitle),
// lastPrompt when no title looks typed, archived, and, on a fork,
// forkedFrom. agents/main/wire.jsonl is the session's own file; each
// subagent writes its own agents/agent-N/wire.jsonl.
//
// state.json's shape: 1.0 and the June/July 1.4 builds keep no id and no
// archived; the session id is then the folder's name. The cwd is workDir
// on those, cwd since version 2.
//
// A fork copies its parent's records byte for byte (calls_kimi.go keys
// them apart), so a fork's wire file is read up to when it was forked,
// as the other agents' readers do with Since.

// kimiSessionFiles are every session's state.json and its wire files: the
// session's own, and its subagents' beside it.
func kimiSessionFiles() []file {
	paths, _ := SessionGlob(filepath.Join(kimiSessions(), "*", "*", "state.json"))
	var out []file
	for _, p := range paths {
		dir := filepath.Dir(p)
		st := readKimiState(p)
		id := cmp.Or(st.ID, filepath.Base(dir))
		if !safeID.MatchString(id) || st.Archived {
			continue
		}
		m := file{agent: "kimi", key: "kimi:" + id, path: p, main: true, manifest: p}
		if stat(&m) {
			out = append(out, m)
		}
		wires, _ := SessionGlob(filepath.Join(dir, "agents", "*", "wire.jsonl"))
		for _, w := range wires {
			f := file{agent: "kimi", key: "kimi:" + id, path: w, main: filepath.Base(filepath.Dir(w)) == "main"}
			if stat(&f) {
				out = append(out, f)
			}
		}
	}
	return out
}

// kimiState is a session's state.json, the fields the page shows.
type kimiState struct {
	ID            string     `json:"id"`
	Cwd           string     `json:"cwd"`
	WorkDir       string     `json:"workDir"`
	CreatedAt     kimiMillis `json:"createdAt"`
	UpdatedAt     kimiMillis `json:"updatedAt"`
	Title         string     `json:"title"`
	IsCustomTitle bool       `json:"isCustomTitle"`
	LastPrompt    string     `json:"lastPrompt"`
	Archived      bool       `json:"archived"`
	ForkedFrom    string     `json:"forkedFrom"`
}

// kimiMillis is a time as Kimi's state keeps it: epoch milliseconds since
// 1.5, an RFC3339 stamp before that (the June and July 1.4 builds).
type kimiMillis int64

func (m *kimiMillis) UnmarshalJSON(b []byte) error {
	if n := int64(0); json.Unmarshal(b, &n) == nil {
		*m = kimiMillis(n)
		return nil
	}
	var s string
	if json.Unmarshal(b, &s) != nil {
		return nil // neither: no time
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		*m = kimiMillis(t.UnixMilli())
	}
	return nil
}

// readKimiState is a session's state.json, empty when there is none or it
// doesn't parse.
func readKimiState(path string) kimiState {
	var st kimiState
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

// kimiSince is when a fork was made, its source's records read up to;
// zero when the state doesn't say.
func kimiSince(created int64) time.Time {
	if created <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(created).Truncate(time.Second)
}

// parseKimi reads one file whole: the session's state, or a wire file's
// usage records — the same records the call reader reads, here as the
// tokens the page sums, dated and counted apart by model.
func parseKimi(f file) *state {
	s := &state{Size: f.size, Mod: f.mod.UnixNano(), ID: strings.TrimPrefix(f.key, "kimi:")}
	if !strings.HasSuffix(f.path, "wire.jsonl") {
		st := readKimiState(f.path)
		if st.ID != "" {
			s.ID = st.ID
		}
		s.Cwd = cmp.Or(st.Cwd, st.WorkDir)
		if st.IsCustomTitle {
			s.Custom = title(st.Title)
		} else {
			s.Named = title(st.Title)
		}
		if s.Named == "" && s.Custom == "" {
			s.First = untagged(st.LastPrompt)
		}
		if st.CreatedAt > 0 {
			s.saw(time.UnixMilli(int64(st.CreatedAt)), f.main)
		}
		if st.UpdatedAt > 0 {
			s.saw(time.UnixMilli(int64(st.UpdatedAt)), f.main)
		}
		if st.ForkedFrom != "" {
			s.Since = kimiSince(int64(st.CreatedAt))
		}
		return s
	}
	// a wire file: the session's own, or a subagent's. A fork repeats its
	// source's records, so they are read only past the fork.
	var since time.Time
	if f.main {
		if st := readKimiState(filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(f.path))), "state.json")); st.ForkedFrom != "" {
			since = kimiSince(int64(st.CreatedAt))
		}
	}
	r, err := os.Open(f.path)
	if err != nil {
		return s
	}
	defer r.Close()
	br := bufio.NewReaderSize(r, 1<<20)
	for {
		b, err := br.ReadBytes('\n')
		if err != nil && (!errors.Is(err, io.EOF) || len(bytes.TrimSpace(b)) == 0) {
			break
		}
		b = bytes.TrimSpace(b)
		if len(b) > 0 && len(b) <= maxLine && bytes.Contains(b, kimiUsageRecord) {
			var rec kimiRecord
			if json.Unmarshal(b, &rec) == nil && rec.Type == "usage.record" && rec.Usage != nil &&
				rec.Model != "" && !strings.HasPrefix(rec.Model, kimiGateway) {
				u := rec.Usage
				if at := ms(rec.Time); at.After(since) && u.InputOther >= 0 && u.Output >= 0 && u.InputCacheRead >= 0 && u.InputCacheCreation >= 0 {
					s.use(dateOf(at), rec.Model, Tokens{Input: u.InputOther, Output: u.Output, CacheRead: u.InputCacheRead, CacheWrite: u.InputCacheCreation})
					s.saw(at, f.main)
				}
			}
		}
		if err != nil {
			break
		}
	}
	return s
}
