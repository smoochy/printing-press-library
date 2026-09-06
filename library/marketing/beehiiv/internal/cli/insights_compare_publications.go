// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Compare Publications: side-by-side table across every synced publication.
// pp:data-source computed

package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newNovelInsightsComparePublicationsCmd(flags *rootFlags) *cobra.Command {
	var flagDB string

	cmd := &cobra.Command{
		Use:     "compare-publications",
		Short:   "Side-by-side growth snapshot across every synced publication",
		Long:    "Use this command for cross-publication side-by-side comparison.\nDo NOT use it for single-publication health; use 'insights growth-summary' instead.",
		Example: "  beehiiv-pp-cli insights compare-publications --agent --select publications.name,publications.subscriber_counts.active",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights compare-publications")
			}
			db, closeDB, ok := insightsStore(cmd, flags, flagDB)
			if !ok {
				return nil
			}
			defer closeDB()
			ctx := cmd.Context()
			pubs := syncedPublications(ctx, db)
			sort.Slice(pubs, func(i, j int) bool {
				a, _ := pubs[i]["name"].(string)
				b, _ := pubs[j]["name"].(string)
				return a < b
			})
			subs, err := scanSubscriptions(ctx, db, "") // cross-publication by design
			if err != nil {
				return usageErr(fmt.Errorf("querying subscriptions: %w", err))
			}
			active := 0
			churned := 0
			for _, s := range subs {
				if s.Status == "unsubscribed" || s.Status == "inactive" {
					churned++
					continue
				}
				active++
			}
			result := map[string]any{
				"publications":          pubs,
				"store_subscriptions":   map[string]any{"active": active, "churned": churned, "total": len(subs)},
				"attribution_note":      "subscriptions sync without publication tags; sync one publication per mirror (separate --db) for exact per-publication attribution",
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}
