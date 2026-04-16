# Test Strategy

Focused tests:

- `go test ./provider/kimi -run 'Test(ParseKimiUsageEvents|CollectKimiUsageEvents)'`
  - Proves new event parsing and collection behavior.
  - Must fail before implementation and pass after implementation.

Provider package tests:

- `go test ./provider/kimi`
  - Proves existing session parser behavior remains compatible.

Repository gates before closing:

- `make fmt`
- `make test`
- `make vet`
- `make build`
- `make lint` when `golangci-lint` is available

Coverage target:

- one event per `StatusUpdate`
- event timestamp equals the status line timestamp
- cross-day records remain separate events
- metadata model/title/session fallback is preserved
- log fallback still populates event model names
