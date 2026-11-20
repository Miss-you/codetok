#!/bin/bash
# Propose one small engineering-cleanup PR (issue #58, docs/auto-optimize.md).
#
#   scripts/auto-optimize.sh [--dry-run | --no-pr]
#
#   --dry-run   Stop after picking a candidate: no branch, no agent, no PR.
#               Runs on the current checkout and never changes it.
#   --no-pr     Do everything except push and open the PR (local rehearsal).
#               The commit stays on a local auto-optimize/* branch.
#
# Steps: open-PR limit -> fixed tools (golangci-lint, deadcode, go test -cover,
# README-vs-cmd flags) -> pick one candidate -> agent change (acpx + agy) ->
# scope/size check + make test + make lint, at most 2 fix rounds -> independent
# review (scripts/ai-review.sh, a different model) -> PR with the label. Every
# decision is made by scripts/autooptimize (Go, tested by `make test`). A run
# that cannot produce a clean change exits 0 without a PR and says why.
#
# Environment:
#   AUTO_OPTIMIZE_LABEL      PR label (default auto-optimize). It must exist.
#   AUTO_OPTIMIZE_MAX_OPEN   Open optimization PRs allowed (default 1).
#   AUTO_OPTIMIZE_MAX_LINES  Diff cap in added+deleted lines (default 200).
#   AUTO_OPTIMIZE_MAX_ROUNDS Fix rounds after the first attempt (default 2).
#   AUTO_OPTIMIZE_MIN_COVERAGE  Coverage threshold for candidates (default 60).
#   AUTO_OPTIMIZE_MODEL      Model for the change agent (default claude-sonnet-5-5).
#                            The reviewer is scripts/ai-review.sh (gemini-4-hydrogen).
#   AUTO_OPTIMIZE_TIMEOUT    Seconds per agent turn (default 1800).
#   AUTO_OPTIMIZE_PROXY      Proxy for agy; falls back to AI_REVIEW_PROXY, HTTPS_PROXY,
#                            then http://127.0.0.1:7890. "none" connects directly.
#   AUTO_OPTIMIZE_OUT_DIR    Logs and reports, outside the checkout
#                            (default $TMPDIR/codetok-auto-optimize/<UTC time>).
#   AUTO_OPTIMIZE_STATE_DIR  Keeps abandoned.txt across runs
#                            (default ~/.local/state/codetok-auto-optimize).
#   AUTO_OPTIMIZE_BASE       Base branch (default main).
#   DEADCODE_VERSION         golang.org/x/tools version for deadcode (default v0.52.0).
#
# The whole script is one function called on the last line, so bash has read
# all of it before any step can change files in the checkout.
set -euo pipefail

main() {
    local mode=pr
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run) mode=dry-run; shift ;;
            --no-pr) mode=no-pr; shift ;;
            -h|--help) sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'; return 0 ;;
            *) die "unknown argument $1" 2 ;;
        esac
    done

    cd "$(dirname "${BASH_SOURCE[0]}")/.."
    repo_root=$(pwd)
    repo_slug=Miss-you/codetok
    label=${AUTO_OPTIMIZE_LABEL:-auto-optimize}
    base_branch=${AUTO_OPTIMIZE_BASE:-main}
    max_lines=${AUTO_OPTIMIZE_MAX_LINES:-200}
    max_rounds=${AUTO_OPTIMIZE_MAX_ROUNDS:-2}
    model=${AUTO_OPTIMIZE_MODEL:-claude-sonnet-5-5}
    agent_timeout=${AUTO_OPTIMIZE_TIMEOUT:-1800}
    state_dir=${AUTO_OPTIMIZE_STATE_DIR:-${HOME}/.local/state/codetok-auto-optimize}
    out=${AUTO_OPTIMIZE_OUT_DIR:-${TMPDIR:-/tmp}/codetok-auto-optimize/$(date -u +%Y%m%dT%H%M%SZ)}
    mkdir -p "$out" "$state_dir"
    out=$(cd "$out" && pwd)
    log "mode $mode, logs in $out"

    for tool in go git gh jq golangci-lint; do need "$tool"; done
    [[ $mode == dry-run ]] || { need acpx; need agy; }

    go_path=$(go env GOPATH) go_cache=$(go env GOCACHE) go_modcache=$(go env GOMODCACHE)
    # A reused out dir must not carry state or reports from an earlier run.
    (cd "$out" && rm -rf rounds guard.diff guard-*.txt failure.md check.txt numstat.txt test.log make-lint.log \
        verdict.txt pr-body.md review-comment.md review review.log agent-*.log prompt-*.md out-of-scope.txt)
    # Bootstrap build for the open-PR gate; rebuilt from the base commit below.
    ao="$out/autooptimize"
    go build -o "$ao" ./scripts/autooptimize

    if [[ $mode != dry-run ]]; then
        [[ -z $(git status --porcelain) ]] || die "the working tree is not clean; refusing to run"
        setup_proxy
    fi
    if [[ $mode == pr ]]; then
        gh label list -R "$repo_slug" --search "$label" --limit 100 --json name --jq '.[].name' | grep -qxF "$label" \
            || die "label '$label' does not exist in $repo_slug; create it first (docs/auto-optimize.md)"
    fi

    # 1. At most N open optimization PRs.
    gh pr list -R "$repo_slug" --label "$label" --state open --json number,url >"$out/open-prs.json"
    gate "$ao" gate -open-prs "$out/open-prs.json" -max-open "${AUTO_OPTIMIZE_MAX_OPEN:-1}"

    # 2. Base commit. A dry run inspects the checkout as it is.
    if [[ $mode == dry-run ]]; then
        base=$(git rev-parse HEAD)
    else
        git fetch -q origin "refs/heads/$base_branch"
        base=$(git rev-parse FETCH_HEAD)
        git checkout -q --detach "$base"
        # The tool comes from the commit under test, built before any agent edit.
        go build -o "$ao" ./scripts/autooptimize
    fi
    log "base $base"

    # 3. Candidates from fixed tools only.
    collect_candidates
    jq -r '.[] | "\(.id) \(.kind) \(.package // "-") \(.message)"' "$out/candidates.json" >"$out/candidates.txt"
    log "$(wc -l <"$out/candidates.txt" | tr -d ' ') candidate(s), list in $out/candidates.txt"

    # 4. Pick one, skipping candidates a person rejected or an earlier run abandoned.
    gh pr list -R "$repo_slug" --label "$label" --state closed --limit 1000 \
        --json number,body,mergedAt >"$out/closed-prs.json"
    gate "$ao" select -candidates "$out/candidates.json" -exclude "$state_dir/abandoned.txt" \
        -closed-prs "$out/closed-prs.json" -out "$out/candidate.json"
    cand_id=$(describe id) cand_title=$(describe title)
    log "picked $cand_id: $(describe kind) at $(describe location): $(describe message)"
    if [[ $mode == dry-run ]]; then
        log "dry run: stopping before the agent"
        return 0
    fi

    branch="auto-optimize/$(date -u +%Y%m%dT%H%M%SZ)-$cand_id"
    git checkout -q -B "$branch"

    # 5. Change, then verify; at most max_rounds fix rounds.
    write_change_prompt >"$out/prompt-0.md"
    run_agent "$out/prompt-0.md" "$out/agent-0.log" || give_up "the change agent failed (see agent-0.log)"
    while true; do
        if verify; then break; fi
        local round status=0
        round=$("$ao" round -state "$out/rounds" -max "$max_rounds") || status=$?
        [[ $status -ne 10 ]] || abandon "checks still fail after $max_rounds fix round(s); see failure.md"
        [[ $status -eq 0 ]] || die "autooptimize round failed with status $status"
        log "fix round $round"
        write_fix_prompt >"$out/prompt-$round.md"
        run_agent "$out/prompt-$round.md" "$out/agent-$round.log" || give_up "the change agent failed in fix round $round"
    done

    git commit -q -m "chore(auto-optimize): $cand_title" -m "Candidate $cand_id: $(describe message)" \
        -m "Proposed by scripts/auto-optimize.sh (#58)."
    head=$(git rev-parse HEAD)

    # 6. Independent review by a different model.
    write_review_inputs
    AI_REVIEW_PROXY=$proxy AI_REVIEW_OUT_DIR="$out/review" \
        scripts/ai-review.sh --what "@$out/review-what.md" --requirements "@$out/review-requirements.md" \
        "$base" "$head" >"$out/review.log" 2>&1 || give_up "scripts/ai-review.sh failed (see review.log)"
    local vstatus=0
    "$ao" verdict -review "$out/review/review.md" >"$out/verdict.txt" 2>&1 || vstatus=$?
    [[ $vstatus -ne 10 ]] || abandon "$(verdict_reason)"
    [[ $vstatus -eq 0 ]] || give_up "autooptimize verdict failed with status $vstatus (see verdict.txt)"
    log "$(verdict_reason)"

    write_pr_body >"$out/pr-body.md"
    write_review_comment >"$out/review-comment.md"
    no_secrets "$out/pr-body.md" "$out/review-comment.md"
    if [[ $mode == no-pr ]]; then
        log "no-pr: change committed on local branch $branch ($head); PR body in $out/pr-body.md"
        return 0
    fi

    # 7. Small PR for a person to review. Never merged by this script.
    git push -q -u origin "$branch"
    pr_url=$(gh pr create -R "$repo_slug" --base "$base_branch" --head "$branch" \
        --title "chore(auto-optimize): $cand_title" --body-file "$out/pr-body.md" --label "$label")
    gh pr comment "$pr_url" -R "$repo_slug" --body-file "$out/review-comment.md" >/dev/null
    log "opened $pr_url"
    summary "Opened $pr_url for candidate \`$cand_id\` ($cand_title)."
}

log() { printf 'auto-optimize: %s\n' "$*" >&2; }
die() { log "error: $1"; summary "Failed: $1"; exit "${2:-1}"; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 is not on PATH" 127; }

# summary appends a line to the GitHub Actions job summary when there is one.
summary() { [[ -z ${GITHUB_STEP_SUMMARY:-} ]] || printf '%s\n' "$*" >>"$GITHUB_STEP_SUMMARY"; }

# gate runs a decision step. Exit 10 means "end the run without a PR".
gate() {
    local status=0
    "$@" 2>"$out/step.err" || status=$?
    cat "$out/step.err" >&2
    if [[ $status -eq 10 ]]; then
        log "no PR this run"
        summary "Skipped, no PR this run: $(tail -n 1 "$out/step.err")"
        exit 0
    fi
    [[ $status -eq 0 ]] || die "$(basename "$1") $2 failed with status $status"
}

describe() { "$ao" describe -candidate "$out/candidate.json" -field "$1"; }

# abandon ends the run without a PR because the change itself failed (checks,
# scope, review), and remembers the candidate so later runs move on.
abandon() {
    printf '%s %s %s\n' "$cand_id" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$1" >>"$state_dir/abandoned.txt"
    end_without_pr "abandoned candidate $cand_id: $1"
}

# give_up ends the run without a PR because of a tool or service failure
# (agent timeout, proxy, reviewer down); the candidate is tried again next run.
give_up() { end_without_pr "no PR for candidate $cand_id, will retry next run: $1"; }

end_without_pr() {
    log "$1"
    summary "No PR: $1"
    git reset -q --hard
    git clean -qfd
    git checkout -q --detach "$base"
    git branch -q -D "$branch" 2>/dev/null || true
    exit 0
}

# guard_snapshot records what the agent must not change: git's config, hooks
# and excludes, the ignored files in the checkout, and this run's own state.
guard_files() {
    local common gitdir
    common=$(git rev-parse --path-format=absolute --git-common-dir)
    gitdir=$(git rev-parse --absolute-git-dir)
    {
        printf '%s\n' "$common/config" "$gitdir/config.worktree"
        find "$common/hooks" "$common/info" -type f 2>/dev/null
        printf '%s\n' "$ao" "$out/candidate.json"
        [[ ! -f $out/rounds ]] || echo "$out/rounds"
    } | sort -u | while IFS= read -r f; do [[ ! -f $f ]] || echo "$f"; done
}

guard_snapshot() {
    local common
    common=$(git rev-parse --path-format=absolute --git-common-dir)
    rm -rf "$out/git-backup" && mkdir -p "$out/git-backup"
    cp -p "$common/config" "$out/git-backup/config"
    local gitdir
    gitdir=$(git rev-parse --absolute-git-dir)
    [[ ! -f $gitdir/config.worktree ]] || cp -p "$gitdir/config.worktree" "$out/git-backup/config.worktree"
    for d in hooks info; do [[ ! -d $common/$d ]] || cp -Rp "$common/$d" "$out/git-backup/$d"; done
    guard_state >"$out/guard-before.txt"
}

guard_state() {
    guard_files | while IFS= read -r f; do shasum -a 256 "$f"; done
    echo "-- ignored --"
    git -c core.quotePath=false status --porcelain --ignored=matching --untracked-files=all | grep '^!!' || true
}

guard_check() {
    guard_state >"$out/guard-after.txt"
    cmp -s "$out/guard-before.txt" "$out/guard-after.txt" && return 0
    diff "$out/guard-before.txt" "$out/guard-after.txt" >"$out/guard.diff" || true
    # Put git's config, hooks and excludes back and delete new ignored files,
    # so nothing the agent planted survives on the runner.
    local common p
    common=$(git rev-parse --path-format=absolute --git-common-dir)
    cp -p "$out/git-backup/config" "$common/config"
    local gitdir
    gitdir=$(git rev-parse --absolute-git-dir)
    rm -f "$gitdir/config.worktree"
    [[ ! -f $out/git-backup/config.worktree ]] || cp -p "$out/git-backup/config.worktree" "$gitdir/config.worktree"
    for d in hooks info; do
        rm -rf "${common:?}/$d"
        [[ ! -d $out/git-backup/$d ]] || cp -Rp "$out/git-backup/$d" "$common/$d"
    done
    sed -n 's/^> !! //p' "$out/guard.diff" | while IFS= read -r p; do
        if [[ $p != /* && $p != *..* ]]; then rm -rf -- "${repo_root:?}/$p"; fi
    done
    give_up "the agent changed git config, hooks, ignored files or run state (see guard.diff)"
}

# no_secrets refuses to publish text that looks like a credential.
no_secrets() {
    # Token prefixes must start a word, so prose like "task-..." does not match.
    local tokens='gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|sk-[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{30,}'
    if grep -n -E "(^|[^A-Za-z0-9_-])($tokens)|-----BEGIN [A-Z ]*PRIVATE KEY" "$@" >/dev/null; then
        give_up "the PR text looks like it contains a credential; nothing was published"
    fi
}

collect_candidates() {
    golangci-lint run --output.json.path="$out/lint.json" --issues-exit-code=0 ./... \
        >"$out/lint.log" 2>&1 || die "golangci-lint failed on the base commit (see lint.log)"
    go run "golang.org/x/tools/cmd/deadcode@${DEADCODE_VERSION:-v0.52.0}" -json ./... >"$out/deadcode.json" \
        2>"$out/deadcode.log" || die "deadcode failed on the base commit (see deadcode.log)"
    # e2e has no statements of its own and takes minutes; make test still runs it.
    # shellcheck disable=SC2046
    go test -cover $(go list ./... | grep -v '/e2e$') >"$out/cover.txt" 2>&1 \
        || die "go test fails on the base commit; fix main first (see cover.txt)"
    "$ao" collect -repo . -lint "$out/lint.json" -deadcode "$out/deadcode.json" -cover "$out/cover.txt" \
        -min-coverage "${AUTO_OPTIMIZE_MIN_COVERAGE:-60}" -out "$out/candidates.json"
}

setup_proxy() {
    proxy=${AUTO_OPTIMIZE_PROXY:-${AI_REVIEW_PROXY:-${HTTPS_PROXY:-${https_proxy:-http://127.0.0.1:7890}}}}
    if [[ $proxy == none ]]; then
        unset HTTPS_PROXY HTTP_PROXY https_proxy http_proxy
    else
        export HTTPS_PROXY=$proxy HTTP_PROXY=$proxy https_proxy=$proxy http_proxy=$proxy
        export NO_PROXY=${NO_PROXY:-localhost,127.0.0.1,::1,.local} no_proxy=${NO_PROXY:-localhost,127.0.0.1,::1,.local}
    fi
    unset ALL_PROXY all_proxy
}

# The change agent may read and edit files only: no terminal, no commands, no
# network fetches. Tests and lint are run by this script, not by the agent.
run_agent() {
    local policy='{"autoApprove":["read","search","edit","think"],"autoDeny":["execute","fetch","delete","move","other"],"defaultAction":"deny"}'
    local status=0
    guard_snapshot
    acpx --cwd "$repo_root" --permission-policy "$policy" --non-interactive-permissions deny --no-terminal \
        --suppress-reads --model "$model" --timeout "$agent_timeout" --format text --agent "$(command -v agy) acp" \
        exec -f "$1" >"$2" 2>&1 || status=$?
    guard_check
    [[ $status -ne 0 ]] || return 0
    # Exit 5: every permission request was denied (usually a shell command).
    # If the turn still ended normally, let the checks judge what was edited.
    if [[ $status -eq 5 ]] && tr -d '\r' <"$2" | grep -v '^[[:space:]]*$' | tail -n 1 | grep -q '^\[done\] end_turn'; then
        log "the agent asked for a denied permission (see $(basename "$2")); continuing with its edits"
        return 0
    fi
    return "$status"
}

# verify stages the change and runs every check; on failure it writes
# failure.md for the next fix prompt. The scope check runs first: a change
# outside the allowed files is never built or run.
verify() {
    : >"$out/failure.md"
    git add -A || die "git add failed"
    local f
    while IFS= read -r f; do gofmt -w "$f" || true; done < <(git -c core.quotePath=false diff --cached --name-only --diff-filter=d -- '*.go')
    git add -A || die "git add failed"
    git -c core.quotePath=false diff --cached --numstat --no-renames "$base" >"$out/numstat.txt" \
        || die "git diff --numstat failed"
    if [[ ! -s $out/numstat.txt ]]; then
        abandon "the agent made no change (it may have judged the candidate unsafe; see the agent log)"
    fi
    if ! "$ao" check -candidate "$out/candidate.json" -numstat "$out/numstat.txt" -max-lines "$max_lines" \
        -out-of-scope "$out/out-of-scope.txt" >"$out/check.txt" 2>&1; then
        { echo "## Scope check failed (tests were not run)"; echo; cat "$out/check.txt"; echo; } >>"$out/failure.md"
        revert_out_of_scope
        return 1
    fi
    local ok=0
    if ! sandboxed make test >"$out/test.log" 2>&1; then
        { echo "## make test failed (last 120 lines)"; echo; echo '```'; tail -n 120 "$out/test.log"; echo '```'; echo; } >>"$out/failure.md"
        ok=1
    fi
    if ! sandboxed make lint >"$out/make-lint.log" 2>&1; then
        { echo "## make lint failed (last 80 lines)"; echo; echo '```'; tail -n 80 "$out/make-lint.log"; echo '```'; echo; } >>"$out/failure.md"
        ok=1
    fi
    return $ok
}

# revert_out_of_scope puts every out-of-scope path back to the base commit
# (or removes it if it is new), since the agent cannot delete files itself.
revert_out_of_scope() {
    [[ -s $out/out-of-scope.txt ]] || return 0
    local p
    while IFS= read -r p; do
        if git cat-file -e "$base:$p" 2>/dev/null; then
            git checkout -q "$base" -- "$p"
        else
            git rm -q -f --cached -- "$p" >/dev/null 2>&1 || true
            rm -f -- "${repo_root:?}/$p"
        fi
    done <"$out/out-of-scope.txt"
    git add -A || die "git add failed"
    {
        echo "The script reverted these out-of-scope files to the base version (new files were deleted):"
        sed 's/^/- /' "$out/out-of-scope.txt"
        echo
    } >>"$out/failure.md"
}

# sandboxed runs the agent's code (tests) with a scrubbed environment: an empty
# HOME, so gh, git and agy logins in the real HOME are not found, no proxy and
# no inherited tokens. It is not a security boundary against hostile code (the
# process still runs as the runner user); it keeps honest test code away from
# credentials. A person reviews every PR before merging.
sandboxed() {
    local home="$out/sandbox-home"
    mkdir -p "$home"
    env -i PATH="$PATH" HOME="$home" TMPDIR="${TMPDIR:-/tmp}" LANG="${LANG:-en_US.UTF-8}" \
        GOPATH="$go_path" GOCACHE="$go_cache" GOMODCACHE="$go_modcache" GOPROXY=off \
        GOLANGCI_LINT_CACHE="$home/.cache/golangci-lint" "$@"
}

verdict_reason() { tail -n 1 "$out/verdict.txt" 2>/dev/null || echo "the reviewer rejected the change"; }

rules() {
    cat <<EOF
Rules (a script checks the first three and discards the change if they fail):
- Change only these files: $(describe allowed). Never touch go.mod, go.sum, Makefile, CI, scripts/ or docs/.
- Keep the diff within $max_lines changed lines (added + deleted). Smaller is better.
- \`make test\` and \`make lint\` must pass. You cannot run commands; the script runs them and sends you any failure.
- Do not change CLI flags, command names, stdout/stderr text, exit codes or JSON output (AGENTS.md "Command Contracts").
- Do not add dependencies, refactor across packages, or fix anything other than this one candidate.
- If the candidate is a false positive or cannot be fixed safely within these rules, change nothing and say why.
EOF
}

write_change_prompt() {
    cat <<EOF
You are making one small, safe cleanup in the codetok repository (a Go CLI) at $repo_root.
Read AGENTS.md and CLAUDE.md first.

$(candidate_block)

$(rules)

You have no shell: do not run commands (grep, go, git, make ...); such requests are denied.
Use your built-in file reading, file search and text search tools instead, then edit the files directly.
When done, reply with a short summary of what you changed and why it is safe.
EOF
}

candidate_block() {
    cat <<EOF
## Candidate $cand_id

- Kind: $(describe kind)
- Where: $(describe location)
- Found by: $(jq -r .source "$out/candidate.json")
- Finding: $(describe message)

$(kind_guidance)
EOF
}

kind_guidance() {
    case $(describe kind) in
        lint) echo "Fix this lint finding the way the linter intends. Do not silence it with nolint comments." ;;
        deadcode) echo "deadcode (run without -test) says nothing reachable from main calls this function; only tests or nothing do. Confirm with a search. If it is truly unused in production, delete it together with tests that exist only for it. Keep anything still used by production code." ;;
        docs) echo "README.md and README_zh.md must describe the flags that cmd/ actually defines. Fix the README text, keeping the two languages consistent. Read cmd/ to learn what the flag does; do not change cmd/." ;;
        coverage) echo "Add focused table-driven tests for untested behavior in this package, following the existing test style and testdata/ fixtures. Do not change non-test code." ;;
    esac
}

write_fix_prompt() {
    cat <<EOF
Your change for candidate $cand_id did not pass the checks. Fix the problems below with the smallest edit.
If the problem cannot be fixed within the rules, revert your change completely.
The files in the repository already contain your previous edits (minus anything the script reverted).

$(candidate_block)

$(cat "$out/failure.md")

$(rules)
EOF
}

write_review_inputs() {
    {
        echo "Automated cleanup for candidate $cand_id: $cand_title."
        echo "Finding ($(jq -r .source "$out/candidate.json"), $(describe location)): $(describe message)"
    } >"$out/review-what.md"
    {
        echo "This is an automated, single-candidate cleanup. Reject it (Critical or Important) if it:"
        echo "- changes behavior: CLI flags, output text, exit codes, JSON fields, token-counting semantics;"
        echo "- goes beyond fixing the one finding above, or touches files outside: $(describe allowed);"
        echo "- deletes code that production paths still use, or weakens tests."
        echo "make test and make lint already passed on this exact commit."
    } >"$out/review-requirements.md"
}

write_pr_body() {
    cat <<EOF
Automated cleanup proposed by \`scripts/auto-optimize.sh\` (#58). **Review before merging; this PR is never merged automatically.**

## Candidate

| | |
| --- | --- |
| ID | \`$cand_id\` |
| Kind | $(describe kind) |
| Found by | $(jq -r .source "$out/candidate.json") |
| Where | \`$(describe location)\` |
| Finding | $(describe message) |

## Change

\`\`\`
$(git -c core.quotePath=false diff --stat "$base" HEAD)
\`\`\`

Allowed scope: $(describe allowed). Cap: $max_lines changed lines.

## Why it is safe

- Only the allowed files changed; the scope and size check passed: $(tail -n 1 "$out/check.txt")
- \`make test\` and \`make lint\` passed on this commit after $(cat "$out/rounds" 2>/dev/null || echo 0) fix round(s) (max $max_rounds).
- Independent review by a different model (\`scripts/ai-review.sh\`, change made by \`$model\`): $(verdict_reason). Full review in the \`AI review:\` comment.

If you reject this PR, close it without merging: later runs skip this candidate.

$("$ao" marker -candidate "$out/candidate.json")
EOF
}

write_review_comment() {
    cat <<EOF
AI review: $head

- Reviewer: \`scripts/ai-review.sh\` (acpx + agy, ${AI_REVIEW_MODEL:-gemini-4-hydrogen}); change agent: \`$model\`
- Range: \`${base:0:7}..${head:0:7}\`
- Command: \`scripts/ai-review.sh --what @review-what.md --requirements @review-requirements.md $base $head\`
- Gate: $(verdict_reason)

<details><summary>Full review</summary>

$(review_answer)

</details>
EOF
}

# review_answer prints the reviewer's answer only, from its first Markdown
# heading or bold title line on, without the acpx trace (tool calls, file reads).
review_answer() {
    awk '/^#{1,6} / || /^\*\*[^*]+\*\*:?[[:space:]]*$/ {on=1}
         on && !/^\[(client|tool|done|thinking)\]/ && !/^  (input|output|kind|files):/' "$out/review/review.md" \
        | relative_paths
}

# relative_paths turns local absolute paths into repo-relative ones, and any
# other path under $HOME into ~/..., so no local user name is published.
relative_paths() {
    sed -e "s#file://${repo_root}/##g" -e "s#${repo_root}/##g" -e "s#file://${HOME}/#~/#g" -e "s#${HOME}/#~/#g"
}

# Sourcing the script (tests) defines the functions without running anything.
[[ ${BASH_SOURCE[0]} != "$0" ]] || main "$@"
