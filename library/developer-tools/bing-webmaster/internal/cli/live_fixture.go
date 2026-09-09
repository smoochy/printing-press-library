package cli

import (
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// PATCH: Supply operator-owned site fixtures without publishing account data.
// This changes only test-discovery metadata, never normal command defaults.
func configureLiveSiteFixtures(root *cobra.Command) {
	site := os.Getenv("BING_WEBMASTER_TEST_SITE")
	u, err := url.Parse(site)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || strings.ContainsAny(site, ";\r\n\t ") {
		return
	}
	// Cover the full discovery tree, not only the quick-matrix commands.
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Flags().Lookup("site") != nil {
			if cmd.Annotations == nil {
				cmd.Annotations = map[string]string{}
			}
			parts := []string{"--site=" + site}
			for _, flag := range []string{"url", "link"} {
				if cmd.Flags().Lookup(flag) != nil {
					parts = append(parts, "--"+flag+"="+site)
				}
			}
			if cmd.Parent() != nil && cmd.Parent().Name() == "traffic" && cmd.Name() != "children-traffic" && cmd.Flags().Lookup("page") != nil {
				parts = append(parts, "--page="+site)
			}
			for flag, env := range map[string]string{"query": "BING_WEBMASTER_TEST_QUERY", "feed-url": "BING_WEBMASTER_TEST_FEED", "gsc": "BING_WEBMASTER_TEST_GSC", "from-sitemap": "BING_WEBMASTER_TEST_FEED"} {
				if value := os.Getenv(env); value != "" && !strings.ContainsAny(value, ";\r\n") && cmd.Flags().Lookup(flag) != nil {
					parts = append(parts, "--"+flag+"="+value)
				}
			}
			cmd.Annotations["pp:happy-args"] = strings.Join(parts, ";")
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
	for _, path := range [][]string{{"drift"}, {"crawl", "children-info"}, {"traffic", "children-traffic"}} {
		cmd, _, err := root.Find(path)
		if err != nil || cmd.Annotations["mcp:read-only"] != "true" {
			continue
		}
		fixture := "--site=" + site
		if len(path) == 2 {
			fixture += ";--url=" + strings.TrimSuffix(u.Host+u.EscapedPath(), "/") + ";--page=0"
		}
		cmd.Annotations["pp:happy-args"] = fixture
	}
}
