package cli

import (
	"github.com/spf13/cobra"
	"strings"
	"testing"
)

func TestFullLiveFixtures(t *testing.T) {
	t.Setenv("BING_WEBMASTER_TEST_SITE", "https://example.org/")
	t.Setenv("BING_WEBMASTER_TEST_GSC", "/fixture/synthetic.csv")
	t.Setenv("BING_WEBMASTER_TEST_FEED", "https://example.org/sitemap.xml")
	root := &cobra.Command{Use: "test"}
	for _, name := range []string{"gap", "publish", "quota"} {
		cmd := &cobra.Command{Use: name}
		cmd.Flags().String("site", "", "site")
		cmd.Flags().String("gsc", "", "CSV")
		cmd.Flags().String("from-sitemap", "", "sitemap")
		root.AddCommand(cmd)
	}
	configureLiveSiteFixtures(root)
	for _, cmd := range root.Commands() {
		got := cmd.Annotations["pp:happy-args"]
		for _, want := range []string{"--site=https://example.org/", "--gsc=/fixture/synthetic.csv", "--from-sitemap=https://example.org/sitemap.xml"} {
			if !strings.Contains(got, want) {
				t.Fatalf("%s missing %s", cmd.Name(), want)
			}
		}
		if cmd.Flags().Lookup("site").DefValue != "" {
			t.Fatal("fixture changed normal defaults")
		}
	}
}

func TestRetiredDeeplinkCommandsAbsent(t *testing.T) {
	cmd := newDeeplinksCmd(&rootFlags{})
	for _, child := range cmd.Commands() {
		if child.Name() == "get" || child.Name() == "algo-urls" {
			t.Fatal("retired command exposed")
		}
	}
}

func TestUnavailableSiteMovesAbsent(t *testing.T) {
	for _, child := range newSitesCmd(&rootFlags{}).Commands() {
		if child.Name() == "moves" {
			t.Fatal("unavailable site moves exposed")
		}
	}
}
