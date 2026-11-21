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

caller_dir=$(pwd)
cd "$(dirname "${BASH_SOURCE[0]}")/.."
repo_slug=${ARCHIVE_REPO:-Miss-you/codetok}

die() { printf 'post-records-comment: %s\n' "$1" >&2; exit "${2:-1}"; }
need_value() { [[ $# -ge 2 ]] || die "$1 needs a value" 2; }

pr='' task='' dry_run=0 tool_args=() paths=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --pr) need_value "$@"; pr=$2; [[ $pr =~ ^[0-9]+$ ]] || die "--pr needs a number" 2; shift 2 ;;
        --task) need_value "$@"; task=$2; shift 2 ;;
        --inline) need_value "$@"
            [[ $2 == ?*=?* ]] || die "--inline needs NAME=FILE, got: $2" 2
            # FILE is relative to the caller's directory, not the repository root.
            name=${2%%=*} file=${2#*=}
            [[ $file == /* ]] || file=$caller_dir/$file
            tool_args+=(--inline "$name=$file"); shift 2 ;;
        --dry-run) dry_run=1; shift ;;
        -h|--help) sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'; exit 0 ;;
        -*) die "unknown option $1" 2 ;;
        *) tool_args+=("$1"); paths+=("$1"); shift ;;
    esac
done
[[ -n $pr && -n $task ]] || die "usage: scripts/post-records-comment.sh --pr <number> --task <task-id> <path>..." 2
# bash 3.2 (macOS /bin/bash) treats an empty array as unset under set -u.
[[ ${#tool_args[@]} -gt 0 ]] || die "list at least one path or --inline NAME=FILE" 2
for bin in go python3; do command -v "$bin" >/dev/null 2>&1 || die "$bin is not on PATH." 127; done

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
body=$work/body.md
go run ./scripts/archive-records comment --task "$task" "${tool_args[@]}" >"$body"

# Listed repository paths must be tracked and committed, because the archive
# reads them from the merge commit. Catch typos and forgotten commits here.
for arg in ${paths[@]+"${paths[@]}"}; do
    if [[ -d $arg && $arg != */ ]]; then die "$arg is a directory; write it as $arg/"; fi
    if [[ -f ${arg%/} && $arg == */ ]]; then die "$arg is a file; drop the trailing /"; fi
    [[ -n $(git ls-files -- "$arg") ]] || die "$arg is not tracked by git; commit it first"
    [[ -z $(git status --porcelain -- "$arg") ]] \
        || printf 'post-records-comment: warning: %s has uncommitted changes; the archive uses the committed version\n' "$arg" >&2
done

if [[ $dry_run -eq 1 ]]; then
    cat "$body"
    exit 0
fi

command -v gh >/dev/null 2>&1 || die "gh is not on PATH." 127
marker="<!-- codetok-records:$task -->"
# Same rule as the archive: the marker on a line of its own, in a comment by an
# owner, member or collaborator. Anything else is not a records comment.
# gh's --jq is gojq (Go regexp), so (?m) makes ^ and $ match at line breaks.
ids=$(gh api --paginate "repos/$repo_slug/issues/$pr/comments" \
    --jq '.[] | select(.author_association == "OWNER" or .author_association == "MEMBER" or .author_association == "COLLABORATOR")
               | select(.body | test("(?m)^<!-- codetok-records:[^ >\r\n]* -->[ \t]*\r?$")) | .id')
count=$(grep -c . <<<"$ids" || true)
if [[ $count -gt 1 ]]; then
    die "PR #$pr already has $count records comments ($(tr '\n' ' ' <<<"$ids")); delete the extras, then rerun"
fi
payload=$work/payload.json
python3 -c 'import json,sys; json.dump({"body": open(sys.argv[1], encoding="utf-8").read()}, open(sys.argv[2], "w", encoding="utf-8"))' "$body" "$payload"
if [[ $count -eq 1 ]]; then
    existing=$(gh api "repos/$repo_slug/issues/comments/$ids" --jq .body)
    tr -d '\r' <<<"$existing" | sed 's/[[:space:]]*$//' | grep -qxF "$marker" || die "PR #$pr has a records comment ($ids) for another task; edit or delete it first"
    url=$(gh api -X PATCH "repos/$repo_slug/issues/comments/$ids" --input "$payload" --jq .html_url)
    printf 'post-records-comment: updated %s\n' "$url" >&2
else
    url=$(gh api -X POST "repos/$repo_slug/issues/$pr/comments" --input "$payload" --jq .html_url)
    printf 'post-records-comment: posted %s\n' "$url" >&2
fi
