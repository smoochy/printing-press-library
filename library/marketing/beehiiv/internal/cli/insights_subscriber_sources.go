// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Subscriber Sources: acquisition attribution grouped from the synced store.
// pp:data-source computed

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newNovelInsightsSubscriberSourcesCmd(flags *rootFlags) *cobra.Command {
	var (
		flagLimit int
		flagDB    string
	)

	cmd := &cobra.Command{
		Use:     "subscriber-sources [publicationId]",
		Short:   "See where new subscribers come from: UTM, channel, and referrer, grouped in one call",
		Long:    "Use this command for total audience acquisition by source.\nDo NOT use it for unsubscribe attribution; use 'insights churn-sources' instead.",
		Example: "  beehiiv-pp-cli insights subscriber-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20 --agent",
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
				return writeDryRun(cmd.OutOrStdout(), flags, "insights subscriber-sources")
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
			ctx := cmd.Context()
			subs, err := scanSubscriptions(ctx, db, pubID)
			if err != nil {
				return usageErr(fmt.Errorf("querying subscriptions: %w", err))
			}
			sources := make([]string, 0, len(subs))
			channels := make([]string, 0, len(subs))
			mediums := make([]string, 0, len(subs))
			sites := make([]string, 0, len(subs))
			combined := make([]string, 0, len(subs))
			active, churned := 0, 0
			for _, s := range subs {
				sources = append(sources, s.UTMSource)
				channels = append(channels, s.UTMChannel)
				mediums = append(mediums, s.UTMMedium)
				sites = append(sites, s.ReferringSite)
				combined = append(combined, strings.Join(nonEmptyParts(s.UTMSource, s.UTMChannel, s.ReferringSite), " / "))
				switch s.Status {
				case "unsubscribed", "inactive":
					churned++
				default:
					active++
				}
			}
			pubs := syncedPublications(ctx, db)
			result := map[string]any{
				"scope_warning": publicationScopeNote(pubs, pubID),
				"publication_id": optionalArg(args),
				"scanned_subscriptions": len(subs),
				"net_growth":            active - churned,
				"active":                active,
				"churned":               churned,
				"sources":               topCounts(countNonEmpty(sources), flagLimit),
				"channels":              topCounts(countNonEmpty(channels), flagLimit),
				"mediums":               topCounts(countNonEmpty(mediums), flagLimit),
				"referring_sites":       topCounts(countNonEmpty(sites), flagLimit),
				"combined":              topCounts(countNonEmpty(combined), flagLimit),
				"publications":          syncedPublications(ctx, db),
				"note":             publicationTagNote(ctx, db, "subscriptions", pubID, len(subs)),
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().IntVar(&flagLimit, "limit", 20, "Maximum entries per grouping")
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}

func nonEmptyParts(parts ...string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = append(out, "(unknown)")
	}
	return out
}
