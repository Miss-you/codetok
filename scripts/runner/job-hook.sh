#!/bin/bash
# Job hook for the self-hosted runner pool (ENG-014, issue #63).
#
# Installed (copied, never run from a checkout) to ~/actions-runner/_shared/hooks/ by
# scripts/runner/setup-pool.sh and wired into every runner instance through .env:
#
#   ACTIONS_RUNNER_HOOK_JOB_STARTED=<shared>/hooks/job-started.sh    -> job-hook.sh started
#   ACTIONS_RUNNER_HOOK_JOB_COMPLETED=<shared>/hooks/job-completed.sh -> job-hook.sh completed
#
# started:   reset the persistent workspace left by the previous job, so every job starts
#            from a clean tree even if the previous job (or its completed hook) crashed.
# completed: reset the workspace again, so build outputs, AI scratch files and any token
#            written into .git/config do not sit on disk until the next job, then prune
#            the shared caches if they are over their size caps.
#
# The .git directory itself is kept: that is what makes the next checkout an incremental
# fetch instead of a full clone.
#
# Logs of this public repository are world-readable: print only short status lines, no
# paths under $HOME and no environment dumps.
set -uo pipefail

phase="${1:-}"
SHARED_DIR="${CODETOK_RUNNER_SHARED_DIR:-$HOME/actions-runner/_shared}"
RUNNERS_ROOT="${CODETOK_RUNNERS_ROOT:-$HOME/actions-runner}"
GOCACHE_MAX_MB="${CODETOK_RUNNER_GOCACHE_MAX_MB:-10240}"
GOMODCACHE_MAX_MB="${CODETOK_RUNNER_GOMODCACHE_MAX_MB:-4096}"
LINTCACHE_MAX_MB="${CODETOK_RUNNER_LINTCACHE_MAX_MB:-2048}"
DIAG_MAX_AGE_DAYS="${CODETOK_RUNNER_DIAG_MAX_AGE_DAYS:-14}"

log() { echo "[runner-hook:${phase}] $*"; }

# under_root PATH: true when PATH is a real directory strictly inside RUNNERS_ROOT.
# Guards every rm -rf below against an empty or unexpected path.
under_root() {
  local p="$1" root
  [ -n "$p" ] && [ -d "$p" ] || return 1
  p=$(cd "$p" && pwd -P) || return 1
  root=$(cd "$RUNNERS_ROOT" 2>/dev/null && pwd -P) || return 1
  case "$p/" in "$root"/?*/) return 0 ;; *) return 1 ;; esac
}

reset_workspace() {
  local ws="${GITHUB_WORKSPACE:-}"
  if [ -z "$ws" ] || [ ! -d "$ws" ]; then
    log "no workspace yet"
    return 0
  fi
  if ! under_root "$ws"; then
    log "workspace is outside the runner root, leaving it alone"
    return 0
  fi
  if [ -d "$ws/.git" ]; then
    # Credentials actions/checkout may have persisted (persist-credentials: true): the
    # http extraheader (checkout v4) or the includeIf credentials include (v6). The
    # checkout post step removes them; this covers a job that died before its post step.
    git -C "$ws" config --local --name-only --get-regexp '^(http\..*\.extraheader|includeif\..*\.path)$' 2>/dev/null |
      while read -r key; do git -C "$ws" config --local --unset-all "$key" || true; done
    git -C "$ws" rebase --abort >/dev/null 2>&1 || true
    git -C "$ws" merge --abort >/dev/null 2>&1 || true
    git -C "$ws" am --abort >/dev/null 2>&1 || true
    if git -C "$ws" rev-parse --verify -q HEAD >/dev/null; then
      git -C "$ws" reset -q --hard HEAD || log "git reset failed"
    fi
    # -ffdx: also untracked nested repositories and ignored files (bin/, coverage, AI output).
    git -C "$ws" clean -q -ffdx || log "git clean failed"
    log "workspace reset (git clean -ffdx, reset --hard; .git kept for incremental fetch)"
  else
    # Not a git checkout (a job that never ran actions/checkout): nothing worth keeping.
    find "$ws" -mindepth 1 -maxdepth 1 -exec rm -rf {} + 2>/dev/null || true
    log "workspace emptied (no git checkout)"
  fi
  # Sibling directories in the same repository folder (checkout `path:` other than the
  # default) are left by design; jobs on this runner check out into the default path.
}

size_mb() { du -sm "$1" 2>/dev/null | awk '{print $1}'; }

# Other runner instances currently executing a job. This hook runs inside our own
# Runner.Worker, so more than one worker process means someone else is busy.
other_jobs_running() {
  local n
  n=$(pgrep -f "$RUNNERS_ROOT/[^/]*/bin/Runner.Worker" 2>/dev/null | wc -l | tr -d ' ')
  [ "${n:-0}" -gt 1 ]
}

prune_dir() { # name dir cap_mb
  local name="$1" dir="$2" cap="$3" size
  [ -d "$dir" ] || return 0
  under_root "$dir" || { log "$name: not under the runner root, skipped"; return 0; }
  size=$(size_mb "$dir")
  if [ "${size:-0}" -le "$cap" ]; then
    log "$name: ${size}MB <= ${cap}MB cap"
    return 0
  fi
  if other_jobs_running; then
    log "$name: ${size}MB > ${cap}MB cap, but another runner is busy; pruning next time"
    return 0
  fi
  chmod -R u+w "$dir" 2>/dev/null || true # module cache files are read-only
  rm -rf "$dir" && mkdir -p "$dir"
  log "$name: ${size}MB > ${cap}MB cap, cleared"
}

prune_caches() {
  local lock="$SHARED_DIR/prune.lock"
  # mkdir is atomic: only one instance prunes at a time. A lock older than an hour is stale.
  if [ -d "$lock" ] && [ -n "$(find "$lock" -maxdepth 0 -mmin +60 2>/dev/null)" ]; then
    rmdir "$lock" 2>/dev/null || true
  fi
  if ! mkdir "$lock" 2>/dev/null; then
    log "another instance is pruning, skipped"
    return 0
  fi
  # GOCACHE also trims itself (entries unused for 5 days); the cap bounds bursts.
  prune_dir GOCACHE "${GOCACHE:-$SHARED_DIR/cache/go-build}" "$GOCACHE_MAX_MB"
  prune_dir GOMODCACHE "${GOMODCACHE:-$SHARED_DIR/cache/go-mod}" "$GOMODCACHE_MAX_MB"
  prune_dir GOLANGCI_LINT_CACHE "${GOLANGCI_LINT_CACHE:-$SHARED_DIR/cache/golangci-lint}" "$LINTCACHE_MAX_MB"
  # Runner diagnostic logs grow by a few files per job.
  local d
  for d in "$RUNNERS_ROOT"/*/_diag; do
    [ -d "$d" ] && find "$d" -type f -name '*.log' -mtime +"$DIAG_MAX_AGE_DAYS" -delete 2>/dev/null
  done
  rmdir "$lock" 2>/dev/null || true
}

case "$phase" in
  started)
    reset_workspace
    ;;
  completed)
    reset_workspace
    prune_caches
    ;;
  *)
    echo "usage: $0 started|completed" >&2
    exit 2
    ;;
esac
# A hook failure fails the job; cleanup problems are logged above instead.
exit 0
