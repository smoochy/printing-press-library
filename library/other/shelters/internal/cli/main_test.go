// Copyright 2026 Abe Diaz (@abe238) and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"os"
	"testing"
)

// TestMain keeps command tests offline now that visibility verification is a
// mandatory part of every shelter-loading path. Focused tests replace this stub
// when they need hidden rows or an error.
func TestMain(m *testing.M) {
	fetchRedCrossHidden = func(context.Context) ([]Shelter, error) {
		return []Shelter{}, nil
	}
	os.Exit(m.Run())
}
