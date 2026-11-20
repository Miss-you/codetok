# ENG-014 Original Implementation (after ENG-011)

- One runner, `codetok-mac-mini`, in `~/actions-runner/codetok`, one LaunchAgent. Jobs from smoke, #57 archive, #58 optimize and AI review queue behind each other.
- `_work` is persistent by default, so `actions/checkout` already reused `.git`; nothing guaranteed a clean tree beyond checkout's own `clean: true`, and nothing cleaned up after a job (build outputs, AI scratch files and a persisted token stayed on disk until the next job of that repo).
- Jobs run as the login user with `HOME=/Users/apple`, so `go` used the maintainer's personal `~/go/pkg/mod` (1.4 GB) and `~/Library/Caches/go-build` (0.7 GB): shared with interactive work, unbounded, and impossible to prune safely from CI.
- Runbook maintenance: "clean `_work` now and then" by hand.
