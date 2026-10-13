package agent

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A provider renamed moves StepCode's magpie/<old>/ entries in enabledModels
// onto the new id, once: the model it is on, another of that provider's the
// user listed, and a glob. The user's own entries stay, a name that only
// shares a prefix stays, and Disconnect still puts the user's scope back
// (those entries renamed) and takes out the one magpie appended.
func TestStepCodeEnabledModelsFollowRename(t *testing.T) {
	home, _ := museHome(t)
	t.Setenv("STEP_CODING_AGENT_DIR", "")
	t.Setenv("STEPCODE_CONFIG_DIR", "")
	dir := filepath.Join(home, ".stepcode")
	configPath := filepath.Join(dir, "config.toml")
	modelsPath := filepath.Join(dir, "models.json")
	const before = `# mine
theme = "dark"
defaultProvider = "anthropic"
defaultModel = "claude"
enabledModels = ["anthropic/*", "custom/fast", "magpie/deepseek/flash", "magpie/deepseek/*", "magpie/deepseekx/flash"]
`
	stepcodeWriteTestFile(t, configPath, before)
	stepcodeWriteTestFile(t, modelsPath, `{"providers":{"anthropic":{"name":"Anthropic"}}}`)

	a := stepcode(home)
	if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	if _, err := RenameProvider("deepseek", "ds"); err != nil {
		t.Fatal(err)
	}
	raw := stepcodeReadTestFile(t, configPath)
	cfg, err := readStepcodeTOML(configPath)
	if err != nil || cfg.Scope == nil {
		t.Fatalf("scope: %v\n%s", err, raw)
	}
	want := []string{"anthropic/*", "custom/fast", "magpie/ds/flash", "magpie/ds/*", "magpie/deepseekx/flash", "magpie/ds/pro"}
	if !slices.Equal(*cfg.Scope, want) || strings.Contains(raw, "magpie/deepseek/") ||
		!strings.Contains(raw, "# mine") || !strings.Contains(raw, `theme = "dark"`) ||
		stepcodeValue(cfg.Provider) != magpieID || stepcodeValue(cfg.Model) != "ds/pro" {
		t.Fatalf("renamed scope = %v\nprovider %q model %q\n%s", *cfg.Scope, stepcodeValue(cfg.Provider), stepcodeValue(cfg.Model), raw)
	}

	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	raw = stepcodeReadTestFile(t, configPath)
	cfg, err = readStepcodeTOML(configPath)
	if err != nil || cfg.Scope == nil {
		t.Fatalf("restored scope: %v\n%s", err, raw)
	}
	restored := []string{"anthropic/*", "custom/fast", "magpie/ds/flash", "magpie/ds/*", "magpie/deepseekx/flash"}
	if !slices.Equal(*cfg.Scope, restored) || strings.Contains(raw, "magpie/deepseek/") || strings.Contains(raw, "magpie/ds/pro") ||
		!strings.Contains(raw, "# mine") || !strings.Contains(raw, `theme = "dark"`) ||
		stepcodeValue(cfg.Provider) != "anthropic" || stepcodeValue(cfg.Model) != "claude" {
		t.Fatalf("disconnect scope = %v\nprovider %q model %q\n%s", *cfg.Scope, stepcodeValue(cfg.Provider), stepcodeValue(cfg.Model), raw)
	}
}

// A rename of a provider StepCode's list does not name leaves that list, and
// a config with no list, as they were. Nothing of the user's is rewritten.
func TestStepCodeRenameLeavesUserScope(t *testing.T) {
	home, _ := museHome(t)
	t.Setenv("STEP_CODING_AGENT_DIR", "")
	t.Setenv("STEPCODE_CONFIG_DIR", "")
	dir := filepath.Join(home, ".stepcode")
	configPath := filepath.Join(dir, "config.toml")
	const withList = `# mine
theme = "dark"
defaultProvider = "anthropic"
defaultModel = "claude"
enabledModels = ["anthropic/*", "custom/fast", "custom/fast"]
`
	stepcodeWriteTestFile(t, configPath, withList)
	if _, err := RenameProvider("deepseek", "ds"); err != nil {
		t.Fatal(err)
	}
	if got := stepcodeReadTestFile(t, configPath); got != withList {
		t.Fatalf("user scope rewritten:\n%s", got)
	}

	const noList = "# mine\ntheme = \"dark\"\ndefaultProvider = \"anthropic\"\ndefaultModel = \"claude\"\n"
	stepcodeWriteTestFile(t, configPath, noList)
	if _, err := RenameProvider("ds", "deepseek"); err != nil {
		t.Fatal(err)
	}
	if got := stepcodeReadTestFile(t, configPath); got != noList {
		t.Fatalf("list created:\n%s", got)
	}
}
