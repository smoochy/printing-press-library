// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Novel command scaffold. Implement the RunE body before shipping.
// generate --force preserves implemented bodies; untouched TODO scaffolds may refresh.
// pp:data-source auto
// Supported strategies: auto, local, live, or computed. Change this default deliberately.

package cli

import (
	"github.com/spf13/cobra"
)

func newNovelInsightsCmd(flags *rootFlags) *cobra.Command {

	cmd := &cobra.Command{
		Use:         "insights",
		Short:       "Growth answers from the local store",
		Example:     "  beehiiv-pp-cli insights churn-sources pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --limit 20 --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	addNovelCommandIfAbsent(cmd, newNovelInsightsChurnSourcesCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsGrowthSummaryCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsComparePublicationsCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsPostPerformanceCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsReferralHealthCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsSendTimesCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsSubscriberLookupCmd(flags))
	addNovelCommandIfAbsent(cmd, newNovelInsightsSubscriberSourcesCmd(flags))
	return cmd
}
