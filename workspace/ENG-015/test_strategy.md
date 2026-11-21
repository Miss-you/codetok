# ENG-015 Test Strategy

| Claim | Evidence |
| --- | --- |
| All e2e tests still pass with a shared binary | `go test -count=1 -v ./e2e/`: 27 `--- PASS`, 0 `--- FAIL` |
| Race-clean, whole repo green | `make test` (`go test -race -cover ./...`) and `go test -race -cover -count=1 ./...` green |
| Build happens once per package run | Wall time drops from ~190 s to ~1 s (one build instead of 27) |
| Inherited GOCACHE is used, and a cold cache still works | `GOCACHE=$(mktemp -d) go test -count=1 ./e2e/` green |
| No other regressions | `make fmt` (no diff), `make vet`, `make build`, `make lint` (0 issues) |
