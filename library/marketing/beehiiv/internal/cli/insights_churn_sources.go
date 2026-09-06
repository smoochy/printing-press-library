// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Churn Sources: unsubscribe attribution grouped from the synced store.
// pp:data-source computed

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newNovelInsightsChurnSourcesCmd(flags *rootFlags) *cobra.Command {
	var (
		flagLimit int
		flagDB    string
	)

	cmd := &cobra.Command{
		Use:     "churn-sources [publicationId]",
		Short:   "See which sources, channels, and campaigns drive unsubscribes",
		Long:    "Use this command for unsubscribe attribution.\nDo NOT use it for total acquisition by source; use 'insights subscriber-sources' instead.",
		Example: "  beehiiv-pp-cli insights churn-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20 --agent",
		Annotations: map[string]string{
			"pp:typed-exit-codes": "0,3",
			"pp:happy-args":       "pub=pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7;--limit=20;--agent",
			"mcp:read-only": "true",
			"pp:data-source": "computed",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights churn-sources")
			}
			db, closeDB, ok := insightsStore(cmd, flags, flagDB)
			if !ok {
				return nil
			}
			defer closeDB()
			pubID := optionalArg(args)
			if pubs := syncedPublications(cmd.Context(), db); len(pubs) > 0 && !publicationInMirror(pubs, pubID) && !beehiivPrefixedIDRE.MatchString(pubID) {
				return notFoundErr(fmt.Errorf("invalid publication id %q", pubID))
			}
			subs, err := scanSubscriptions(cmd.Context(), db, pubID)
			if err != nil {
				return usageErr(fmt.Errorf("querying subscriptions: %w", err))
			}
			churned := 0
			sources := map[string]int{}
			channels := map[string]int{}
			campaigns := map[string]int{}
			sites := map[string]int{}
			combined := map[string]int{}
			for _, s := range subs {
				if s.Status != "unsubscribed" && s.Status != "inactive" {
					continue
				}
				churned++
				bump(sources, s.UTMSource)
				bump(channels, s.UTMChannel)
				bump(campaigns, s.UTMCampaign)
				bump(sites, s.ReferringSite)
				key := strings.Join(nonEmptyParts(s.UTMSource, s.UTMChannel, s.ReferringSite), " / ")
				bump(combined, key)
			}
			pubs := syncedPublications(cmd.Context(), db)
			result := map[string]any{
				"scope_warning": publicationScopeNote(pubs, pubID),
				"publication_id": optionalArg(args),
				"scanned_subscriptions": len(subs),
				"churned":               churned,
				"sources":               topCounts(sources, flagLimit),
				"channels":              topCounts(channels, flagLimit),
				"campaigns":             topCounts(campaigns, flagLimit),
				"referring_sites":       topCounts(sites, flagLimit),
				"combined":              topCounts(combined, flagLimit),
			}
			if churned == 0 {
				result["note"] = publicationTagNote(cmd.Context(), db, "subscriptions", pubID, len(subs))
				if result["note"] == "" {
					result["note"] = "no unsubscribed subscribers in the local mirror"
				}
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().IntVar(&flagLimit, "limit", 20, "Maximum entries per grouping")
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}

func bump(m map[string]int, key string) {
	if key == "" {
		key = "(unknown)"
	}
	m[key]++
}
