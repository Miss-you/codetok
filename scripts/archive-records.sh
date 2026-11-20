#!/bin/bash
# Archive the process records of a merged PR into docs/changes/<task-id>/ on main.
#
#   scripts/archive-records.sh [options] --pr <number>
#
# Run by .github/workflows/archive-records.yml on the self-hosted runner after a
# PR merges, or by hand from a clean checkout of this repository.
#
# Steps:
#   1. Read the PR and its comments with gh. A PR from a fork, a PR that is not
#      merged into main, or a PR without a records comment
#      (<!-- codetok-records:<task-id> --> by an owner, member or collaborator)
#      is skipped: exit 0, no commit.
#   2. Copy every listed file verbatim from the merge commit, plus inline blocks
#      from the comment, into a staging directory (scripts/archive-records plan).
#      When docs/changes/<task-id>/manifest.json on main already has the same
#      input digest, stop here: exit 0, no AI call, no commit.
#   3. Ask agy (through acpx, read-only, no terminal) for a summary that matches
#      a fixed JSON schema. The AI never writes files.
#   4. Validate the summary (scripts/archive-records finalize). Invalid output
#      fails the run and writes nothing.
#   5. Commit "docs: archive <task-id> records (#<pr>) [skip ci]" in a temporary
#      worktree at origin/main and push it to main with the machine's own git
#      credentials. No token is read or printed by this script.
#
# Options:
#   --pr <number>   The merged PR to archive (required).
#   --no-push       Commit in the temporary worktree but do not push; print the
#                   worktree path and keep it.
#   --dry-run       Stop after step 2 and print the prompt path and acpx command.
#
# Environment:
#   ARCHIVE_REPO          GitHub repository (default Miss-you/codetok).
#   ARCHIVE_REMOTE        Git remote to fetch from and push to (default origin).
#   ARCHIVE_BRANCH        Branch that receives the archive (default main).
#   AI_REVIEW_PROXY       Proxy for agy, same meaning as in scripts/ai-review.sh.
#                         Default: $HTTPS_PROXY, else http://127.0.0.1:7890; "none"
#                         connects directly.
#   ARCHIVE_MODEL         ACP model id (default gemini-4-hydrogen).
#   ARCHIVE_THOUGHT       thought_level config option (default high).
#   ARCHIVE_TIMEOUT       Seconds to wait for agy (default 900).
#   ARCHIVE_RECORDS_BIN   Prebuilt scripts/archive-records binary (tests use it).
set -euo pipefail

script_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
repo_slug=${ARCHIVE_REPO:-Miss-you/codetok}
remote=${ARCHIVE_REMOTE:-origin}
branch=${ARCHIVE_BRANCH:-main}
dest=docs/changes

die() { printf 'archive-records: %s\n' "$1" >&2; exit "${2:-1}"; }
say() { printf 'archive-records: %s\n' "$1" >&2; }
need_value() { [[ $# -ge 2 ]] || die "$1 needs a value" 2; }

pr='' push=1 dry_run=0
while [[ $# -gt 0 ]]; do
    case $1 in
        --pr) need_value "$@"; pr=$2; [[ $pr =~ ^[0-9]+$ ]] || die "--pr needs a number" 2; shift 2 ;;
        --no-push) push=0; shift ;;
        --dry-run) dry_run=1; shift ;;
        -h|--help) sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'; exit 0 ;;
        *) die "unknown argument $1" 2 ;;
    esac
done
[[ -n $pr ]] || die "usage: scripts/archive-records.sh [--no-push] [--dry-run] --pr <number>" 2

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is not on PATH." 127; }
need gh; need git; need python3
repo_dir=$(git rev-parse --show-toplevel 2>/dev/null) || die "run this from a checkout of $repo_slug" 2

work=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/archive-records.XXXXXX")
worktree=''
cleanup() {
    if [[ -n $worktree ]]; then
        git -C "$repo_dir" worktree remove --force "$worktree" >/dev/null 2>&1 || true
    fi
    rm -rf "$work"
}
trap cleanup EXIT

tool=${ARCHIVE_RECORDS_BIN:-}
if [[ -z $tool ]]; then
    need go
    tool=$work/archive-records
    (cd "$script_root" && go build -o "$tool" ./scripts/archive-records) || die "cannot build scripts/archive-records"
fi

# 1. PR and comments.
gh pr view "$pr" -R "$repo_slug" \
    --json number,title,body,url,state,baseRefName,isCrossRepository,mergeCommit,mergedAt >"$work/pr.json" \
    || die "cannot read PR $pr with gh"
gh api --paginate "repos/$repo_slug/issues/$pr/comments" >"$work/comments.json" \
    || die "cannot read the comments of PR $pr with gh"

git -C "$repo_dir" fetch -q "$remote" "refs/heads/$branch" || die "cannot fetch $branch from $remote"
base_tip=$(git -C "$repo_dir" rev-parse FETCH_HEAD)
merge_sha=$(python3 -c 'import json,sys; m=json.load(open(sys.argv[1])).get("mergeCommit") or {}; print(m.get("oid") or "")' "$work/pr.json")
if [[ -n $merge_sha ]] && ! git -C "$repo_dir" cat-file -e "$merge_sha^{commit}" 2>/dev/null; then
    git -C "$repo_dir" fetch -q "$remote" "$merge_sha" 2>/dev/null || true
fi

# The archive is written into a fresh worktree at the branch tip, so the
# caller's checkout (the runner's workspace or a developer's tree) is untouched.
worktree=$work/tree
git -C "$repo_dir" worktree add -q --detach "$worktree" "$base_tip" || die "cannot create a worktree at $branch"

if [[ -n $merge_sha ]] && git -C "$repo_dir" cat-file -e "$merge_sha^{commit}" 2>/dev/null; then
    # First parent: the base for merge and squash commits alike. A root commit
    # is diffed against the empty tree.
    parent=$(git -C "$repo_dir" rev-parse -q --verify "$merge_sha^1" 2>/dev/null) \
        || parent=$(git -C "$repo_dir" hash-object -t tree /dev/null)
    git -C "$repo_dir" diff --stat=200 "$parent" "$merge_sha" >"$work/diff.patch"
    git -C "$repo_dir" -c core.quotePath=false diff -M "$parent" "$merge_sha" >>"$work/diff.patch"
else
    : >"$work/diff.patch"
fi

# 2. Plan: decide, copy verbatim, write manifest and prompt.
mkdir -p "$work/stage"
"$tool" plan --pr-json "$work/pr.json" --comments "$work/comments.json" --repo "$worktree" \
    --dest "$dest" --stage "$work/stage" --diff "$work/diff.patch" >"$work/plan.out"
plan_value() { sed -n "s/^$1=//p" "$work/plan.out"; }
status=$(plan_value status) task_id=$(plan_value task_id) reason=$(plan_value reason)
if [[ $status == skip ]]; then
    say "PR #$pr skipped: $reason"
    exit 0
fi
[[ $status == archive && -n $task_id ]] || die "unexpected plan result: $(tr '\n' ' ' <"$work/plan.out")"
say "PR #$pr: archiving $task_id into $dest/$task_id/"

# 3. Summary from agy through acpx.
proxy=${AI_REVIEW_PROXY:-${HTTPS_PROXY:-${https_proxy:-http://127.0.0.1:7890}}}
if [[ $proxy == none ]]; then
    unset HTTPS_PROXY HTTP_PROXY https_proxy http_proxy ALL_PROXY all_proxy
else
    if [[ $proxy =~ ^[a-z0-9]+://([^/@]+@)?(\[[0-9a-fA-F:]+\]|[^/:@]+):([0-9]+)/?$ ]]; then
        proxy_host=${BASH_REMATCH[2]} proxy_port=${BASH_REMATCH[3]}
        proxy_host=${proxy_host#\[}; proxy_host=${proxy_host%\]}
    else
        die "AI_REVIEW_PROXY must look like http://host:port, got: $(sed -E 's#://[^/@]+@#://***@#' <<<"$proxy")" 2
    fi
    if [[ $dry_run -eq 0 ]]; then
        need nc
        nc -z -w 3 "$proxy_host" "$proxy_port" >/dev/null 2>&1 \
            || die "proxy $proxy_host:$proxy_port is not reachable. Start the proxy client, or set AI_REVIEW_PROXY (\"none\" to connect directly)."
    fi
    unset ALL_PROXY all_proxy
    export HTTPS_PROXY=$proxy HTTP_PROXY=$proxy https_proxy=$proxy http_proxy=$proxy
    export NO_PROXY=${NO_PROXY:-localhost,127.0.0.1,::1,.local} no_proxy=${NO_PROXY:-localhost,127.0.0.1,::1,.local}
fi

stage_dir=$work/stage
if [[ $dry_run -eq 1 ]]; then
    stage_dir=$(mktemp -d "${TMPDIR:-/tmp}/archive-records-dry-run.XXXXXX")
    cp -R "$work/stage/." "$stage_dir/"
fi
need acpx; need agy
cmd=(acpx --cwd "$stage_dir" --deny-all --non-interactive-permissions deny --no-terminal
     --model "${ARCHIVE_MODEL:-gemini-4-hydrogen}" --timeout "${ARCHIVE_TIMEOUT:-900}" --format text
     --agent "$(command -v agy) acp"
     exec --config-option "thought_level=${ARCHIVE_THOUGHT:-high}" -f "$stage_dir/prompt.md")
if [[ $dry_run -eq 1 ]]; then
    say "dry run: staged archive and prompt in $stage_dir"
    printf '%q ' "${cmd[@]}"; printf '\n'
    exit 0
fi
ai_output=$work/ai-output.txt
ai_status=0
"${cmd[@]}" >"$ai_output" 2>&1 || ai_status=$?
if [[ $ai_status -ne 0 ]]; then
    tail -n 20 "$ai_output" >&2 || true
    die "acpx exited with status $ai_status; nothing was written. If agy cannot reach its API, see docs/agents/faq/ai-review.md."
fi

# 4. Validate and write.
"$tool" finalize --repo "$worktree" --dest "$dest" --stage "$work/stage" --ai-output "$ai_output" >"$work/final.out" || {
    printf 'archive-records: last 40 lines of the agent output:\n' >&2
    tail -n 40 "$ai_output" >&2
    die "the agy summary was rejected; nothing was written or committed."
}
message=$(sed -n 's/^commit_message=//p' "$work/final.out")

# 5. Commit and push.
git -C "$worktree" add -A -- "$dest/$task_id"
if git -C "$worktree" diff --cached --quiet; then
    say "$dest/$task_id is already up to date; nothing to commit."
    exit 0
fi
git -C "$worktree" commit -q -m "$message"
say "committed: $message"
if [[ $push -eq 0 ]]; then
    say "--no-push: commit $(git -C "$worktree" rev-parse --short HEAD) left in worktree $worktree"
    trap - EXIT
    exit 0
fi
for attempt in 1 2 3; do
    if git -C "$worktree" push -q "$remote" "HEAD:refs/heads/$branch"; then
        say "pushed $(git -C "$worktree" rev-parse --short HEAD) to $branch"
        exit 0
    fi
    [[ $attempt -lt 3 ]] || break
    say "push rejected, rebasing on $remote/$branch (attempt $attempt)"
    git -C "$worktree" fetch -q "$remote" "refs/heads/$branch" && git -C "$worktree" rebase -q FETCH_HEAD \
        || die "rebase onto $branch failed"
done
die "could not push to $branch after 3 attempts"
