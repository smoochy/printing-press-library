// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Growth Summary (absorbed feature, rebuilt on the enlarged store).
// pp:data-source computed

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelInsightsGrowthSummaryCmd(flags *rootFlags) *cobra.Command {
	var (
		flagDB    string
		flagLimit int
	)

	cmd := &cobra.Command{
		Use:     "growth-summary <publicationId>",
		Short:   "Summarize publication, subscriber, post, and referral health in one read-only call",
		Long:    "Use this command for the account-level snapshot.\nDo NOT use it for per-post detail; use 'insights post-performance' instead.",
		Example: "  beehiiv-pp-cli insights growth-summary pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:data-source": "computed"},
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights growth-summary")
			}
			db, closeDB, ok := insightsStore(cmd, flags, flagDB)
			if !ok {
				return nil
			}
			defer closeDB()
			ctx := cmd.Context()
			pubs := syncedPublications(ctx, db)
			var pub map[string]any
			if len(args) > 0 {
				for _, p := range pubs {
					if id, _ := p["id"].(string); id == args[0] {
						pub = p
						break
					}
				}
				if pub == nil {
					return notFoundErr(fmt.Errorf("publication %s not found in the local mirror; sync publications first", args[0]))
				}
			}
			subs, err := scanSubscriptions(ctx, db, args[0])
			if err != nil {
				return usageErr(fmt.Errorf("querying subscriptions: %w", err))
			}
			statusCounts := map[string]int{}
			tierCounts := map[string]int{}
			for _, s := range subs {
				bump(statusCounts, s.Status)
				bump(tierCounts, s.Tier)
			}
			postRows, err := scanRows(ctx, db,
				`SELECT id, data FROM posts ORDER BY COALESCE(json_extract(data,'$.publish_date'), json_extract(data,'$.created')) DESC LIMIT ?`,
				flagLimit)
			if err != nil {
				return usageErr(fmt.Errorf("querying posts: %w", err))
			}
			recentPosts := make([]map[string]any, 0, len(postRows))
			for _, r := range postRows {
				m := r.Map()
				recentPosts = append(recentPosts, map[string]any{
					"id": r.ID, "title": firstString(m, "title", "subtitle", "slug"), "status": m["status"],
				})
			}
			fieldRows, _ := scanRows(ctx, db, `SELECT id, data FROM custom_fields`)
			fieldKinds := map[string]int{}
			for _, r := range fieldRows {
				m := r.Map()
				kind, _ := m["kind"].(string)
				bump(fieldKinds, kind)
			}
			segmentRows, _ := scanRows(ctx, db, `SELECT id, data FROM segments`)
			result := map[string]any{
				"publication":     pub,
				"publications_synced": len(pubs),
				"subscribers": map[string]any{
					"total":  len(subs),
					"status": topCounts(statusCounts, 10),
					"tiers":  topCounts(tierCounts, 10),
				},
				"posts": map[string]any{
					"recent_count": len(recentPosts),
					"recent":       recentPosts,
				},
				"custom_fields": map[string]any{"count": len(fieldRows), "kinds": topCounts(fieldKinds, 10)},
				"segments":      map[string]any{"count": len(segmentRows)},
			}
			if len(pubs) > 1 && pub != nil {
				result["note"] = "subscriber and post counts span all synced publications"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	cmd.Flags().IntVar(&flagLimit, "limit", 10, "Maximum recent posts to include")
	return cmd
}
