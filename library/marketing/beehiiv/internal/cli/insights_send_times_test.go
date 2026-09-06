// Copyright 2026 Kevin Magnan and contributors. Licensed under Apache-2.0. See LICENSE.
// cli-printing-press: novel-scaffold-test
// Novel command scaffold tests. Keep the wiring smoke test and add behavior cases as needed.

package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestNovelInsightsSendTimesHelpWires smoke-tests that the insights send-times command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelInsightsSendTimesHelpWires(t *testing.T) {
	cmd := RootCmd()
	cmd.SetArgs([]string{"insights", "send-times", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("insights send-times --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "send-times"} {
		if !strings.Contains(help, want) {
			t.Fatalf("insights send-times --help missing %q in output:\n%s", want, help)
		}
	}
}
