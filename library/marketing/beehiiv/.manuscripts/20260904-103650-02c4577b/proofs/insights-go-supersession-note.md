Reconciliation decision: library internal/cli/insights.go (live-API insight commands,
v4.2.2-era patch code) is intentionally superseded. The approved reprint manifest
rebuilds all six prior insight commands as eight store-based computed commands in the
fresh tree (insights.go + insights_*_solution files). Patch substance that lived in
other files (escapePathParam in helpers.go/sync.go, cache permissions, JWT policy,
dependent sync paths) is preserved by regen-merge WITH-ADDITIONS/NOVEL verdicts.
