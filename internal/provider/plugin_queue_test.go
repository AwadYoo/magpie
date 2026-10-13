package provider

import (
	"testing"

	"github.com/yetone/magpie/internal/plugin"
)

// An extra plugin account (AlsoOn) carries none of the provider's settings:
// the queue bounds the user set on the provider are read from
// providers.json then, as the concurrency limit and the RPM are (#1899:
// queueWait=1 held on the first account, the extra one queued 62s).
func TestExtraPluginAccountTakesTheQueues(t *testing.T) {
	claudeHome(t)
	if err := store(file{Providers: []Provider{{ID: "plugco", Name: "Plug Co", QueueLimit: 1, QueueWait: 1, MaxRPM: 30}}}); err != nil {
		t.Fatal(err)
	}
	p := pluginProvider(plugin.Provider{ID: "plugco", Name: "Plug Co", Models: []plugin.Model{{ID: "m"}}}, pluginLogin{})
	if p.Account == nil {
		t.Fatal("no account")
	}
	if got := p.QueueLimitOf(); got != 1 { // the provider is kept under the id the plugin answers by
		t.Fatalf("queue limit = %d, want the provider's 1", got)
	}
	if got := p.QueueWaitOf(); got != 1 {
		t.Fatalf("queue wait = %d, want the provider's 1", got)
	}
	// the first account, whose provider object the settings were saved on
	// (here the same shape), reads them from itself
	first := pluginProvider(plugin.Provider{ID: "plugco", Name: "Plug Co", Models: []plugin.Model{{ID: "m"}}}, pluginLogin{})
	first.QueueLimit, first.QueueWait = 2, 3
	if got := first.QueueLimitOf(); got != 2 {
		t.Fatalf("own queue limit = %d, want its own 2", got)
	}
	if got := first.QueueWaitOf(); got != 3 {
		t.Fatalf("own queue wait = %d, want its own 3", got)
	}
	// no provider saved: nothing to inherit
	q := pluginProvider(plugin.Provider{ID: "nobody", Name: "Nobody", Models: []plugin.Model{{ID: "m"}}}, pluginLogin{})
	if got := q.QueueLimitOf(); got != 0 {
		t.Fatalf("unkept limit = %d, want 0", got)
	}
	if got := q.QueueWaitOf(); got != 0 {
		t.Fatalf("unkept wait = %d, want 0", got)
	}
}
