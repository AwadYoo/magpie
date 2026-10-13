package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `magpie opencode a/m` wires OpenCode to magpie; `magpie opencode default`
// puts back the model OpenCode was on before (#1900), as the Agents page's
// Disconnect does.
func TestAgentOpenCodeDefaultPutsBackTheModel(t *testing.T) {
	groupsHome(t)
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "opencode.json")
	own := `{
  "$schema": "https://opencode.ai/config.json",
  "model": "native/glm-5",
  "small_model": "native/glm-5-mini",
  "theme": "dark"
}`
	if err := os.WriteFile(path, []byte(own), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func() string {
		b, _ := os.ReadFile(path)
		return string(b)
	}
	if err := run([]string{"opencode", "a/m"}); err != nil {
		t.Fatal(err)
	}
	s := read()
	if !strings.Contains(s, `"model": "magpie/a/m"`) || !strings.Contains(s, `"provider"`) {
		t.Fatalf("not wired:\n%s", s)
	}
	if err := run([]string{"opencode", "default"}); err != nil {
		t.Fatal(err)
	}
	s = read()
	if !strings.Contains(s, `"model": "native/glm-5"`) || !strings.Contains(s, `"small_model": "native/glm-5-mini"`) || strings.Contains(s, `"magpie"`) || strings.Contains(s, "baseURL") || !strings.Contains(s, `"theme": "dark"`) {
		t.Fatalf("default didn't put back what was there:\n%s", s)
	}
}
