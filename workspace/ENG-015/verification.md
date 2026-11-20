# ENG-015 Verification

Machine: Apple Silicon Mac, 10 CPUs, `go1.26.0 darwin/arm64`. Date: 2026-11-21 CST.

## `go test -count=1 ./e2e/` timing

| Run | Go-reported | `time -p` real |
| --- | --- | --- |
| Before (per-test build, fresh GOCACHE each) | 190.481 s | 191.40 s (user 698.52 s) |
| After, warm inherited cache | 1.194 s | 1.78 s |
| After, warm inherited cache (repeat) | 0.830 s | 1.03 s |
| After, cold cache (`GOCACHE=$(mktemp -d)` for the whole run) | 1.304 s | 9.01 s (includes compiling the test binary and deps once) |

## Gates (after the change)

- `go test -count=1 -v ./e2e/`: 27 PASS, 0 FAIL
- `make fmt`: no diff besides the change
- `make vet`: clean
- `make test`: green, `real 3.41 s` (other packages' results partly cached)
- `go test -race -cover -count=1 ./...`: green, `real 2.47 s`, e2e 2.039 s
- `make build`: ok
- `make lint`: 0 issues
