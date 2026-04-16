# EBTA-009 Verification

Fresh verification after final fixture/test edits:

```bash
make fmt
# go fmt ./...
```

```bash
go test -count=1 ./e2e -run TestEventBasedCrossDayAcceptance
# ok  	github.com/miss-you/codetok/e2e	11.585s
```

```bash
go test -count=1 ./e2e -run Test
# ok  	github.com/miss-you/codetok/e2e	217.855s
```

```bash
make test
# go test -race -cover ./...
# ok  	github.com/miss-you/codetok/cmd
# ok  	github.com/miss-you/codetok/cursor
# ok  	github.com/miss-you/codetok/e2e
# ok  	github.com/miss-you/codetok/provider
# ok  	github.com/miss-you/codetok/provider/claude
# ok  	github.com/miss-you/codetok/provider/codex
# ok  	github.com/miss-you/codetok/provider/cursor
# ok  	github.com/miss-you/codetok/provider/kimi
# ok  	github.com/miss-you/codetok/stats
```

```bash
make vet
# go vet ./...
```

```bash
make build
# go build ... -o bin/codetok .
```

```bash
make lint
# 0 issues.
```

The RED pass before fixture creation failed as expected:

```bash
go test ./e2e -run TestEventBasedCrossDayAcceptance
# daily returned []
# session returned []
```
