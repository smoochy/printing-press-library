// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// Send-Times: open-rate evidence by send weekday and hour, from local history.
// pp:data-source computed

package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

type sendSlot struct {
	Weekday string  `json:"weekday"`
	Hour    int     `json:"hour"`
	Sends   int     `json:"sends"`
	OpenRate float64 `json:"avg_open_rate"`
	Rated   int     `json:"rated_sends"`
}

func newNovelInsightsSendTimesCmd(flags *rootFlags) *cobra.Command {
	var flagDB string

	cmd := &cobra.Command{
		Use:         "send-times [publicationId]",
		Short:       "Find your best send slot: open rate by weekday and hour from your own history",
		Example:     "  beehiiv-pp-cli insights send-times pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7 --agent",
		Annotations: map[string]string{ "pp:typed-exit-codes": "0,3", "pp:happy-args": "<publicationId>=pub_477b0b68-0ab1-4b3f-954e-d1f6302b58a7;--agent","mcp:read-only": "true", "pp:data-source": "computed"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return writeDryRun(cmd.OutOrStdout(), flags, "insights send-times")
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
			rows, err := scanRows(cmd.Context(), db, `SELECT id, data FROM posts WHERE 1=1`+pubFilter, pubArgs...)
			if err != nil {
				return usageErr(fmt.Errorf("querying posts: %w", err))
			}
			slots := map[string]*sendSlot{}
			sendsWithoutDate := 0
			ratedTotal := 0
			for _, r := range rows {
				m := r.Map()
				sec := numberField(m, "publish_date", "sent_at", "created")
				weekday, hour, okSlot := unixSlot(int64(sec))
				if !okSlot {
					sendsWithoutDate++
					continue
				}
				key := fmt.Sprintf("%s %02d", weekday, hour)
				slot, exists := slots[key]
				if !exists {
					slot = &sendSlot{Weekday: weekday, Hour: hour}
					slots[key] = slot
				}
				slot.Sends++
				if rate, okRate := openRate(m); okRate {
					slot.OpenRate = (slot.OpenRate*float64(slot.Rated) + rate) / float64(slot.Rated+1)
					slot.Rated++
					ratedTotal++
				}
			}
			list := make([]sendSlot, 0, len(slots))
			for _, s := range slots {
				list = append(list, *s)
			}
			sort.Slice(list, func(i, j int) bool {
				if list[i].Sends != list[j].Sends {
					return list[i].Sends > list[j].Sends
				}
				return list[i].Weekday < list[j].Weekday
			})
			pubs := syncedPublications(cmd.Context(), db)
			result := map[string]any{
				"scope_warning": publicationScopeNote(pubs, pubID),
				"publication_id":          optionalArg(args),
				"slots":                   list,
				"posts_without_send_time": sendsWithoutDate,
				"rated_sends":             ratedTotal,
			}
			if ratedTotal == 0 {
				result["note"] = "no open-rate stats in the mirror; sync posts with expand=stats to rate each slot"
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagDB, "db", "", "Path to the local SQLite mirror")
	return cmd
}

// numberField returns the first non-zero numeric field (json numbers arrive
// as float64 through map decoding).
func numberField(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		switch v := m[k].(type) {
		case float64:
			if v != 0 {
				return v
			}
		case int64:
			if v != 0 {
				return float64(v)
			}
		}
	}
	return 0
}

// openRate reads the open rate from a stats object when the mirror carries it.
func openRate(m map[string]any) (float64, bool) {
	stats, ok := m["stats"].(map[string]any)
	if !ok {
		return 0, false
	}
	switch v := stats["open_rate"].(type) {
	case float64:
		return v, true
	}
	// Derive from opens and recipients when the API returns counts instead.
	opens, okO := stats["opens"].(float64)
	recips, okR := stats["recipients"].(float64)
	if okO && okR && recips > 0 {
		return opens / recips, true
	}
	return 0, false
}
