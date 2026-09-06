// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Referral Health: referral config vs real subscriber code coverage, offline.
// pp:data-source computed

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelInsightsReferralHealthCmd(flags *rootFlags) *cobra.Command {
	var flagDB string

	cmd := &cobra.Command{
		Use:         "referral-health [publicationId]",
		Short:       "Check referral-program config and how many subscribers carry referral codes",
		Example:     "  beehiiv-pp-cli insights referral-health pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent",
		Annotations: map[string]string{ "pp:typed-exit-codes": "0,3", "pp:happy-args": "<publicationId>=pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7;--agent","mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights referral-health")
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
			pubs := syncedPublications(ctx, db)
			referral := map[string]any{}
			rows, err := scanRows(ctx, db, `SELECT id, data FROM referral_program`)
			if err == nil {
				for _, r := range rows {
					referral = r.Map()
					referral["id"] = r.ID
					break
				}
			}
			subs, err := scanSubscriptions(ctx, db, pubID)
			if err != nil {
				return usageErr(fmt.Errorf("querying subscriptions: %w", err))
			}
			withCodes := 0
			for _, s := range subs {
				if s.ReferralCode != "" {
					withCodes++
				}
			}
			result := map[string]any{
				"scope_warning": publicationScopeNote(pubs, pubID),
				"publication_id": optionalArg(args),
				"referral_program":          referral,
				"subscribers_with_codes":    withCodes,
				"scanned_subscriptions":     len(subs),
				"coverage":                  coverageRatio(withCodes, len(subs)),
				"publications":              pubs,
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}

func coverageRatio(part, total int) any {
	if total == 0 {
		return nil
	}
	return float64(part) / float64(total)
}
