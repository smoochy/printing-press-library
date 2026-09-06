// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Post Performance: compact per-send review from the synced posts table.
// pp:data-source computed

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelInsightsPostPerformanceCmd(flags *rootFlags) *cobra.Command {
	var (
		flagLimit int
		flagDB    string
	)

	cmd := &cobra.Command{
		Use:     "post-performance [publicationId]",
		Short:   "Review recent sends with status, timing, and expanded stats in one table",
		Long:    "Use this command for per-post detail.\nDo NOT use it for the account-level snapshot; use 'insights growth-summary' instead.",
		Example: "  beehiiv-pp-cli insights post-performance pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 10 --agent",
		Annotations: map[string]string{ "pp:typed-exit-codes": "0,3", "pp:happy-args": "<publicationId>=pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7;--limit=10;--agent",
			"mcp:read-only": "true",
			"pp:data-source": "computed",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights post-performance")
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
			pubFilter, pubArgs := publicationDataFilter(pubID)
			rows, err := scanRows(cmd.Context(), db,
				`SELECT id, data FROM posts WHERE 1=1`+pubFilter+` ORDER BY COALESCE(json_extract(data,'$.publish_date'), json_extract(data,'$.created')) DESC LIMIT ?`,
				append(pubArgs, flagLimit)...)
			if err != nil {
				return usageErr(fmt.Errorf("querying posts: %w", err))
			}
			posts := make([]map[string]any, 0, len(rows))
			for _, r := range rows {
				m := r.Map()
				post := map[string]any{
					"id":       r.ID,
					"title":    firstString(m, "title", "subtitle", "slug"),
					"status":   m["status"],
					"audience": m["audience"],
					"stats":    m["stats"],
				}
				if v, okm := m["publish_date"]; okm {
					post["publish_date"] = v
				}
				if v, okm := m["created"]; okm {
					post["created"] = v
				}
				posts = append(posts, post)
			}
			pubs := syncedPublications(cmd.Context(), db)
			result := map[string]any{
				"scope_warning": publicationScopeNote(pubs, pubID),
				"note":           publicationTagNote(cmd.Context(), db, "posts", pubID, len(posts)),
				"publication_id": optionalArg(args),
				"posts": posts,
				"stats_note":     "stats are present when the mirror was synced with expand=stats",
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().IntVar(&flagLimit, "limit", 25, "Maximum posts to list")
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
