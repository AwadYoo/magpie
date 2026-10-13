package netproxy

import (
	"net/url"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// The settings status must not display credentials; requests still need them.
func TestDescribeHidesProxyCredentials(t *testing.T) {
	for _, source := range []string{"settings", "environment", "system"} {
		for _, tc := range []struct{ name, raw, want string }{
			{"plain", "socks5://127.0.0.1:1080", "socks5://127.0.0.1:1080"},
			{"authenticated", "socks5://alice:secret@127.0.0.1:1080", "socks5://127.0.0.1:1080"},
			{"encoded", "socks5h://ali%40ce:p%3Aa%2Fss%25@[::1]:1080", "socks5h://[::1]:1080"},
			{"username", "http://alice@127.0.0.1:7890", "http://127.0.0.1:7890"},
		} {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy", "NO_PROXY", "no_proxy"} {
					t.Setenv(k, "")
				}
				sysCache.at, sysCache.p = farFuture, Proxy{}
				t.Cleanup(func() { sysCache.at = zero })
				switch source {
				case "settings":
					if err := settings.Save(settings.Settings{Proxy: tc.raw}); err != nil {
						t.Fatal(err)
					}
				case "environment":
					t.Setenv("ALL_PROXY", tc.raw)
				case "system":
					sysCache.p.URL = tc.raw
				}
				if got, from := Describe(); got != tc.want || from != source {
					t.Fatalf("description = %q (%s), want %q (%s)", got, from, tc.want, source)
				}
				u, _ := url.Parse("https://vendor.example/request")
				if p, err := For(u); err != nil || p == nil || p.String() != tc.raw {
					t.Fatalf("description changed the proxy used by requests: %v", err)
				}
			})
		}
	}
}

// A proxy address magpie logs or shows carries no credentials, whatever
// shape the user typed it in (#1886).
func TestRedactKeepsCredentialsOutOfAProxyAddress(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{"plain", "socks5://127.0.0.1:1080", "socks5://127.0.0.1:1080"},
		{"bare host", "127.0.0.1:7890", "127.0.0.1:7890"},
		{"authenticated", "http://alice:s3cret@10.0.0.2:8080", "http://10.0.0.2:8080"},
		{"socks authenticated", "socks5://alice:s3cret@10.0.0.2:1080", "socks5://10.0.0.2:1080"},
		{"encoded", "http://ali%40ce:p%3Aa%2Fss%25@10.0.0.2:8080", "http://10.0.0.2:8080"},
		{"username only", "http://alice@127.0.0.1:7890", "http://127.0.0.1:7890"},
		{"bare host with userinfo", "alice:s3cret@10.0.0.2:8080", "10.0.0.2:8080"},
		{"malformed", "http://alice:s3cret%zz@10.0.0.2:8080", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Redact(tc.raw); got != tc.want {
				t.Fatalf("Redact(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestDescribeHidesMalformedSystemProxy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy", "NO_PROXY", "no_proxy"} {
		t.Setenv(k, "")
	}
	sysCache.at, sysCache.p = farFuture, Proxy{URL: "socks5://alice:secret%zz@127.0.0.1:1080"}
	t.Cleanup(func() { sysCache.at = zero })
	if got, from := Describe(); got != "" || from != "system" {
		t.Fatalf("malformed proxy description = %q (%s)", got, from)
	}
}
