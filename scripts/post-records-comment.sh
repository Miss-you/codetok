#!/bin/bash
# Post or update the one process-records comment on a PR (#57 handoff).
#
#   scripts/post-records-comment.sh [options] --pr <number> --task <task-id> <path>...
#
# Each <path> is a repository path under workspace/, openspec/changes/ or
# docs/plans/; end a directory with "/" to include every file in it. After the
# PR merges, .github/workflows/archive-records.yml copies them verbatim into
# docs/changes/<task-id>/raw/.
#
# Options:
#   --inline NAME=FILE   Attach a local file that is not in the repository
#                        verbatim, in a collapsed block (repeatable). Check it for
#                        credentials first: the comment is public.
#   --dry-run            Print the comment instead of posting it.
#
# The comment carries the marker <!-- codetok-records:<task-id> -->. When a
# comment with that marker already exists on the PR, it is edited in place, so
# the PR keeps exactly one. This does not commit anything or change the PR head.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
repo_slug=${ARCHIVE_REPO:-Miss-you/codetok}

die() { printf 'post-records-comment: %s\n' "$1" >&2; exit "${2:-1}"; }
need_value() { [[ $# -ge 2 ]] || die "$1 needs a value" 2; }

pr='' task='' dry_run=0 tool_args=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --pr) need_value "$@"; pr=$2; [[ $pr =~ ^[0-9]+$ ]] || die "--pr needs a number" 2; shift 2 ;;
        --task) need_value "$@"; task=$2; shift 2 ;;
        --inline) need_value "$@"; tool_args+=(--inline "$2"); shift 2 ;;
        --dry-run) dry_run=1; shift ;;
        -h|--help) sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'; exit 0 ;;
        -*) die "unknown option $1" 2 ;;
        *) tool_args+=("$1"); shift ;;
    esac
done
[[ -n $pr && -n $task ]] || die "usage: scripts/post-records-comment.sh --pr <number> --task <task-id> <path>..." 2

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
body=$work/body.md
go run ./scripts/archive-records comment --task "$task" "${tool_args[@]}" >"$body"

# Listed repository paths must exist in this checkout, so typos fail here and
# not after the merge.
for arg in "${tool_args[@]}"; do
    case $arg in --inline|*=*) continue ;; esac
    [[ -e $arg ]] || die "$arg does not exist in this checkout"
done

if [[ $dry_run -eq 1 ]]; then
    cat "$body"
    exit 0
fi

command -v gh >/dev/null 2>&1 || die "gh is not on PATH." 127
marker="<!-- codetok-records:$task -->"
ids=$(gh api --paginate "repos/$repo_slug/issues/$pr/comments" \
    --jq ".[] | select(.body | contains(\"<!-- codetok-records:\")) | .id")
count=$(grep -c . <<<"$ids" || true)
if [[ $count -gt 1 ]]; then
    die "PR #$pr already has $count records comments ($(tr '\n' ' ' <<<"$ids")); delete the extras, then rerun"
fi
payload=$work/payload.json
python3 -c 'import json,sys; json.dump({"body": open(sys.argv[1]).read()}, open(sys.argv[2], "w"))' "$body" "$payload"
if [[ $count -eq 1 ]]; then
    existing=$(gh api "repos/$repo_slug/issues/comments/$ids" --jq .body)
    grep -qF "$marker" <<<"$existing" || die "PR #$pr has a records comment ($ids) for another task; edit or delete it first"
    url=$(gh api -X PATCH "repos/$repo_slug/issues/comments/$ids" --input "$payload" --jq .html_url)
    printf 'post-records-comment: updated %s\n' "$url" >&2
else
    url=$(gh api -X POST "repos/$repo_slug/issues/$pr/comments" --input "$payload" --jq .html_url)
    printf 'post-records-comment: posted %s\n' "$url" >&2
fi
