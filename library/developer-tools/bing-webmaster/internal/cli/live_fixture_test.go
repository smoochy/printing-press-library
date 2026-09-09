package cli

import (
	"github.com/spf13/cobra"
	"testing"
)

func TestLiveSiteFixtures(t *testing.T) {
	for _, site := range []string{"", "https://example.org/", "https://example.org/;--confirm=true", "https://user:secret@example.org/", "file:///tmp/site"} {
		t.Run(site, func(t *testing.T) {
			t.Setenv("BING_WEBMASTER_TEST_SITE", site)
			root := &cobra.Command{Use: "test"}
			drift := &cobra.Command{Use: "drift", Annotations: map[string]string{"mcp:read-only": "true"}}
			crawl := &cobra.Command{Use: "crawl"}
			child := &cobra.Command{Use: "children-info", Annotations: map[string]string{"mcp:read-only": "true"}}
			crawl.AddCommand(child)
			root.AddCommand(drift, crawl)
			configureLiveSiteFixtures(root)
			want := ""
			if site == "https://example.org/" {
				want = "--site=" + site
			}
			if got := drift.Annotations["pp:happy-args"]; got != want {
				t.Fatalf("got %q want %q", got, want)
			}
			if want != "" && child.Annotations["pp:happy-args"] != want+";--url=example.org;--page=0" {
				t.Fatal("missing child fixture")
			}
		})
	}
}
