#!/bin/bash
# Set up the self-hosted runner pool on the maintainer's Mac (ENG-014, issue #63).
#
#   scripts/runner/setup-pool.sh [--count N] [--dry-run]
#
# Instance 1 is the runner from ENG-011 (~/actions-runner/codetok, codetok-mac-mini).
# Instances 2..N are ~/actions-runner/codetok-<i> (codetok-mac-mini-<i>), each with its own
# registration, _work directory and launchd LaunchAgent, and the same labels.
#
# For every instance the script writes a managed block into .env with:
#   - the proxy variables (copied from instance 1),
#   - GOMODCACHE / GOCACHE / GOLANGCI_LINT_CACHE under ~/actions-runner/_shared/cache,
#   - ACTIONS_RUNNER_HOOK_JOB_STARTED / _COMPLETED pointing at ~/actions-runner/_shared/hooks.
# Hooks are copied out of this checkout, so a later job cannot change what the hooks run.
#
# Idempotent: re-running it only refreshes hooks and .env blocks and restarts idle
# instances whose .env changed. It never restarts an instance that is running a job.
# The registration token is fetched with the local `gh` login and never printed.
set -euo pipefail

REPO="Miss-you/codetok"
LABELS="codetok-local,ai"
ROOT="$HOME/actions-runner"
PRIMARY="$ROOT/codetok"
PRIMARY_NAME="codetok-mac-mini"
SHARED="$ROOT/_shared"
COUNT=3
DRY_RUN=0
SRC_DIR="$(cd "$(dirname "$0")" && pwd)"

while [ $# -gt 0 ]; do
  case "$1" in
    --count) COUNT="$2"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    *) echo "usage: $0 [--count N] [--dry-run]" >&2; exit 2 ;;
  esac
done

run() {
  echo "+ $*"
  if [ "$DRY_RUN" = 0 ]; then "$@"; fi
}

instance_dir() { if [ "$1" = 1 ]; then echo "$PRIMARY"; else echo "$ROOT/codetok-$1"; fi; }
instance_name() { if [ "$1" = 1 ]; then echo "$PRIMARY_NAME"; else echo "$PRIMARY_NAME-$1"; fi; }

runner_busy() { # name -> prints true/false/unknown
  gh api "repos/$REPO/actions/runners" --paginate \
    --jq ".runners[] | select(.name == \"$1\") | .busy" 2>/dev/null || echo unknown
}

[ -f "$PRIMARY/.runner" ] || { echo "instance 1 ($PRIMARY) is not configured; set it up first (docs/self-hosted-runner.md)" >&2; exit 1; }

# 1. Shared hooks and caches.
run mkdir -p "$SHARED/hooks" "$SHARED/cache/go-build" "$SHARED/cache/go-mod" "$SHARED/cache/golangci-lint"
run install -m 0755 "$SRC_DIR/job-hook.sh" "$SHARED/hooks/job-hook.sh"
for phase in started completed; do
  wrapper="$SHARED/hooks/job-$phase.sh"
  echo "+ write $wrapper"
  if [ "$DRY_RUN" = 0 ]; then
    printf '#!/bin/bash\nexec "%s/hooks/job-hook.sh" %s\n' "$SHARED" "$phase" >"$wrapper"
    chmod 0755 "$wrapper"
  fi
done

# 2. The managed .env block, identical for every instance.
proxy_lines=$(grep -E '^(HTTPS?_PROXY|https?_proxy|NO_PROXY|no_proxy)=' "$PRIMARY/.env" || true)
managed_block() {
  echo "# >>> codetok runner pool (ENG-014), managed by scripts/runner/setup-pool.sh"
  [ -n "$proxy_lines" ] && echo "$proxy_lines"
  echo "GOMODCACHE=$SHARED/cache/go-mod"
  echo "GOCACHE=$SHARED/cache/go-build"
  echo "GOLANGCI_LINT_CACHE=$SHARED/cache/golangci-lint"
  echo "ACTIONS_RUNNER_HOOK_JOB_STARTED=$SHARED/hooks/job-started.sh"
  echo "ACTIONS_RUNNER_HOOK_JOB_COMPLETED=$SHARED/hooks/job-completed.sh"
  echo "# <<< codetok runner pool (ENG-014)"
}

# write_env DIR -> exit 0 when .env changed
write_env() {
  local env="$1/.env" tmp
  tmp=$(mktemp)
  if [ -f "$env" ]; then
    # Drop the old managed block and the proxy lines it now carries.
    awk '/^# >>> codetok runner pool/{skip=1} !skip{print} /^# <<< codetok runner pool/{skip=0}' "$env" |
      grep -vE '^(HTTPS?_PROXY|https?_proxy|NO_PROXY|no_proxy)=' >"$tmp" || true
  fi
  managed_block >>"$tmp"
  if [ -f "$env" ] && cmp -s "$tmp" "$env"; then rm -f "$tmp"; return 1; fi
  echo "+ update $env (managed block)"
  if [ "$DRY_RUN" = 0 ]; then mv "$tmp" "$env"; else diff -u "$env" "$tmp" || true; rm -f "$tmp"; fi
  return 0
}

# 3. Instances.
for i in $(seq 1 "$COUNT"); do
  dir=$(instance_dir "$i")
  name=$(instance_name "$i")
  echo "== instance $i: $name"
  new=0
  if [ ! -f "$dir/.runner" ]; then
    new=1
    # Same runner build as instance 1; no download. Registration state, logs and work dir stay per instance.
    run mkdir -p "$dir"
    run rsync -a --delete \
      --exclude '/.runner' --exclude '/.credentials' --exclude '/.credentials_rsaparams' \
      --exclude '/.env' --exclude '/.path' --exclude '/.service' --exclude '/runsvc.sh' \
      --exclude '/_diag/' --exclude '/_work/' "$PRIMARY/" "$dir/"
    echo "+ (cd $dir && ./config.sh --unattended --url https://github.com/$REPO --token *** --name $name --labels $LABELS --work _work)"
    if [ "$DRY_RUN" = 0 ]; then
      token=$(gh api -X POST "repos/$REPO/actions/runners/registration-token" --jq .token)
      (cd "$dir" && ./config.sh --unattended --url "https://github.com/$REPO" --token "$token" \
        --name "$name" --labels "$LABELS" --work _work)
      unset token
    fi
  fi

  changed=0
  if write_env "$dir"; then changed=1; fi

  svc="actions.runner.${REPO/\//-}.$name"
  plist="$HOME/Library/LaunchAgents/$svc.plist"
  if [ ! -f "$plist" ]; then
    run sh -c "cd '$dir' && ./svc.sh install"
    # Same fix as ENG-011: without SessionCreate the job can read the login keychain (agy).
    run sh -c "/usr/libexec/PlistBuddy -c 'Delete :SessionCreate' '$plist' || true"
    run sh -c "cd '$dir' && ./svc.sh start"
  else
    # A restart skipped earlier (busy instance) is remembered, so a re-run still does it.
    pending="$dir/.pool-restart-pending"
    if [ "$changed" = 1 ] || [ "$new" = 1 ]; then
      [ "$DRY_RUN" = 0 ] && touch "$pending"
    fi
    if [ -f "$pending" ] || { [ "$DRY_RUN" = 1 ] && [ "$changed" = 1 ]; }; then
      busy=$(runner_busy "$name")
      # The busy flag can lag by a few seconds behind job assignment; the job's
      # started hook makes a killed job's leftovers harmless for the next one.
      if [ "$busy" = false ]; then
        run sh -c "cd '$dir' && ./svc.sh stop && ./svc.sh start"
        [ "$DRY_RUN" = 0 ] && rm -f "$pending"
      else
        echo "!! $name is busy=$busy: not restarting now; re-run this script later to restart it"
      fi
    fi
  fi
done

echo "== pool"
gh api "repos/$REPO/actions/runners" --paginate \
  --jq '.runners[] | "\(.name)\t\(.status)\tbusy=\(.busy)\t\([.labels[].name] | join(","))"'
