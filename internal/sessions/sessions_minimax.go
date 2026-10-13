package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

// MiniMax Code's sessions for the Sessions page (#1913): its calls are read
// for Usage already (calls_minimax.go), and the same files hold what this
// page shows. Each session is a folder
// v2/sessions/<yyyy>/<mm>/<dd>/<hh-mm-ss-ms>-session_<base64 of its id>/,
// its messages in messages.jsonl, one per line. A session keeps no state of
// its own: its id is the folder's (minimaxSessionID), its times the
// messages' timestamps in milliseconds, its title the first prompt that
// isn't a system reminder, its cwd nowhere. An assistant message names the
// provider and model that answered and its usage (pi's ai package counts):
// a call through magpie's entry (custom_provider:magpie) is the gateway's
// to count, as the call reader says.
//
// The history-catalog.json beside a messages.jsonl keeps older generations
// of it (messages*.jsonl); a reply copied into another generation keeps its
// message_id, and a message is counted once.

// minimaxSessionFiles are each session's messages files, the call reader's
// own list.
func minimaxSessionFiles() []file { return minimaxCallFiles() }

// parseMiniMax reads a session's messages whole: the times, the title and
// the models' tokens the page sums.
func parseMiniMax(f file) *state {
	s := &state{Size: f.size, Mod: f.mod.UnixNano(), ID: strings.TrimPrefix(f.key, "minimax-code:")}
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
		if len(b) > 0 && len(b) <= maxLine {
			var l mcodeLine
			if json.Unmarshal(b, &l) == nil {
				minimaxSessionLine(s, l)
			}
		}
		if err != nil {
			break
		}
	}
	return s
}

// minimaxSessionLine is one message as the page sees it: an assistant's the
// tokens of its call, a user's the session's first prompt.
func minimaxSessionLine(s *state, l mcodeLine) {
	m := l.Message
	if m == nil {
		return
	}
	if m.Role == "assistant" {
		if u := m.Usage; u != nil && m.Provider != mcodeGateway && m.Model != "" {
			if at := ms(m.Timestamp); !at.IsZero() {
				s.saw(at, true)
				s.use(dateOf(at), m.Model, u.tokens())
			}
		}
		return
	}
	if m.Role == "user" && s.Title == "" && s.First == "" {
		if p := minimaxPrompt(m.Content); p != "" && !strings.HasPrefix(p, "<") {
			// a system reminder isn't something the user said: it sets
			// neither a title nor a first prompt, so a session of
			// reminders and no call of its own is left unlisted
			s.Title = title(p)
			s.saw(ms(m.Timestamp), true)
		}
	}
}

// minimaxPrompt is the text of a user message's content, a string or parts.
func minimaxPrompt(c json.RawMessage) string {
	var s string
	if json.Unmarshal(c, &s) == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(c, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}
	return ""
}
