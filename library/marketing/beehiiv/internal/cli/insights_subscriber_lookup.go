// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Subscriber Lookup: one subscriber by email or subscription ID, offline.
// pp:data-source computed

package cli

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelInsightsSubscriberLookupCmd(flags *rootFlags) *cobra.Command {
	var flagDB string

	cmd := &cobra.Command{
		Use:     "subscriber-lookup [publicationId] <email-or-subscription-id>",
		Short:   "Find one subscriber by email or subscription ID and get a compact record",
		Long:    "Use this command for one subscriber by email or ID.\nDo NOT use it for source-attribution counts; use 'insights subscriber-sources' instead.",
		Example: "  beehiiv-pp-cli insights subscriber-lookup reader@example.com --agent --select subscription.email,subscription.status",
		Annotations: map[string]string{ "pp:happy-args": "<email>=reader@example.com;--agent",
			"mcp:read-only":      "true",
			"pp:data-source":     "computed",
			"pp:typed-exit-codes": "0,3",
		},
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights subscriber-lookup")
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("missing required argument: email or subscription ID"))
			}
			db, closeDB, ok := insightsStore(cmd, flags, flagDB)
			if !ok {
				return nil
			}
			defer closeDB()
			query := args[len(args)-1]
			row := db.DB().QueryRowContext(cmd.Context(),
				`SELECT id, COALESCE(email,''), COALESCE(status,''), COALESCE(subscription_tier,''),
				        COALESCE(referral_code,''), COALESCE(referring_site,''), created, data
				 FROM subscriptions WHERE email = ? OR id = ?`, query, query)
			var (
				id, email, status, tier, code, site string
				created                             sql.NullInt64
				data                                string
			)
			if err := row.Scan(&id, &email, &status, &tier, &code, &site, &created, &data); err != nil {
				if err == sql.ErrNoRows {
					fmt.Fprintf(cmd.ErrOrStderr(), "no subscriber matching %q in the local mirror\n", query)
					if !wantsHumanTable(cmd.OutOrStdout(), flags) {
						_ = printJSONFiltered(cmd.OutOrStdout(), map[string]any{"found": false, "query": query}, flags)
					}
					return notFoundErr(fmt.Errorf("no subscriber matching %q", query))
				}
				return usageErr(fmt.Errorf("querying subscriber: %w", err))
			}
			sub := map[string]any{
				"id": id, "email": email, "status": status, "subscription_tier": tier,
				"referral_code": code, "referring_site": site,
			}
			if created.Valid {
				sub["created"] = created.Int64
			}
			result := map[string]any{"found": true, "publication_id": optionalArg(args[:len(args)-1]), "subscription": sub}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}
