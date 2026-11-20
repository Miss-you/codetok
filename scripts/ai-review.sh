#!/bin/bash
# Run a read-only code review of a Git range with Gemini 4 Hydrogen (High)
# through acpx and the Antigravity CLI (agy).
#
#   scripts/ai-review.sh [options] <base> [<head>]
#   scripts/ai-review.sh [options] --pr <number>
#
# Options:
#   --pr <number>            Review a Miss-you/codetok PR: base is the merge base
#                            with its base branch, head is its head commit, and the PR
#                            title and body fill the description.
#   --what <text|@file>      What was implemented (default: PR title or commit subjects).
#   --requirements <text|@file>
#                            Plan or acceptance criteria to check against
#                            (default: PR body, or "Not provided").
#   --dry-run                Write the prompt and print the acpx command without running it.
#
# Environment:
#   AI_REVIEW_PROXY     Proxy for agy. Default: $HTTPS_PROXY, else http://127.0.0.1:7890.
#                       Set to "none" to connect directly.
#   AI_REVIEW_TEMPLATE  Prompt template. Default: the superpowers
#                       requesting-code-review/code-reviewer.md, searched in
#                       ~/.agents/skills/superpowers, ~/.codex/skills and the
#                       Claude Code plugin cache.
#   AI_REVIEW_MODEL     ACP model id (default gemini-4-hydrogen).
#   AI_REVIEW_THOUGHT   thought_level config option (default high).
#   AI_REVIEW_TIMEOUT   Seconds to wait for the answer (default 1200).
#   AI_REVIEW_OUT_DIR   Where prompt.md and review.md go
#                       (default output/ai-review/<base>-<head>).
#
# Why the options look like this:
# - agy reads only the HTTPS_PROXY and HTTP_PROXY variables, not the macOS system
#   proxy. Sessions that never source ~/.zshrc (env -i, launchd, GUI apps) connect
#   directly; from an unsupported region the model list does not load and agy
#   reports the model as unknown or times out. When the proxy's exit node is in an
#   unsupported region, the API answers "User location is not supported".
# - `acpx antigravity` looks for agy_acp_server.par, which agy does not ship, so the
#   agent is started as `agy acp`. Through ACP the model id has no effort suffix;
#   High is the thought_level option. `gemini-4-hydrogen-high` works only with
#   `agy --model`.
# - The review is read-only: reads are approved, everything else is denied, and the
#   terminal capability is not advertised. Without --no-terminal the model asks to
#   run commands, the request is denied, and acpx aborts with PERMISSION_DENIED.
# - agy retries a transient API error by itself and acpx still prints
#   "⚠ API error (attempt N)". When acpx exits 0 and the output ends with
#   "[done] end_turn", the review finished, so "⚠" lines only produce a warning.
set -euo pipefail

caller_dir=$(pwd)
cd "$(dirname "${BASH_SOURCE[0]}")/.."
repo_root=$(pwd)
repo_slug=Miss-you/codetok

die() { printf 'ai-review: %s\n' "$1" >&2; exit "${2:-1}"; }
# @file paths are relative to the caller's directory, not the repository root.
arg_text() { if [[ $1 == @* ]]; then (cd "$caller_dir" && cat "${1#@}"); else printf '%s' "$1"; fi; }

need_value() { [[ $# -ge 2 ]] || die "$1 needs a value" 2; }

pr='' what='' requirements='' dry_run=0 positional=()
while [[ $# -gt 0 ]]; do
    case $1 in
        --pr) need_value "$@"; pr=$2; [[ $pr =~ ^[0-9]+$ ]] || die "--pr needs a number" 2; shift 2 ;;
        --what) need_value "$@"; what=$(arg_text "$2"); shift 2 ;;
        --requirements) need_value "$@"; requirements=$(arg_text "$2"); shift 2 ;;
        --dry-run) dry_run=1; shift ;;
        -h|--help) sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'; exit 0 ;;
        -*) die "unknown option $1" 2 ;;
        *) positional+=("$1"); shift ;;
    esac
done

need() { command -v "$1" >/dev/null 2>&1 || die "$1 is not on PATH." 127; }
need acpx; need agy; need python3
[[ -z $pr ]] || need gh

# Proxy: agy ignores the macOS system proxy, so always pass one explicitly. Set it
# before any network call, since gh and git may need it too.
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
            || die "proxy $proxy_host:$proxy_port is not reachable. Start the proxy client, or set AI_REVIEW_PROXY to a working proxy (\"none\" to connect directly)."
    fi
    unset ALL_PROXY all_proxy
    export HTTPS_PROXY=$proxy HTTP_PROXY=$proxy https_proxy=$proxy http_proxy=$proxy
    export NO_PROXY=${NO_PROXY:-localhost,127.0.0.1,::1,.local} no_proxy=${NO_PROXY:-localhost,127.0.0.1,::1,.local}
fi

if [[ -n $pr ]]; then
    [[ ${#positional[@]} -eq 0 ]] || die "use either --pr or <base> [<head>], not both" 2
    pr_json=$(gh pr view "$pr" -R "$repo_slug" --json baseRefName,headRefOid,title,body) \
        || die "cannot read PR $pr with gh"
    pr_field() { python3 -c 'import json,sys; print(json.load(sys.stdin).get(sys.argv[1]) or "")' "$1" <<<"$pr_json"; }
    base_branch=$(pr_field baseRefName)
    head=$(pr_field headRefOid)
    # Fetch from the repository URL, not "origin": in a fork checkout origin has no PR refs.
    repo_url=https://github.com/$repo_slug.git
    git fetch -q "$repo_url" "refs/heads/$base_branch" || die "cannot fetch $base_branch from $repo_url"
    base_tip=$(git rev-parse FETCH_HEAD)
    git fetch -q "$repo_url" "refs/pull/$pr/head" || die "cannot fetch PR $pr from $repo_url"
    base=$(git merge-base "$base_tip" "$head") \
        || die "cannot find a merge base between $base_branch and the PR head"
    [[ -n $what ]] || what="PR #$pr: $(pr_field title)"
    [[ -n $requirements ]] || requirements=$(pr_field body)
else
    [[ ${#positional[@]} -ge 1 && ${#positional[@]} -le 2 ]] \
        || die "usage: scripts/ai-review.sh [options] <base> [<head>] | --pr <number>" 2
    base=$(git rev-parse --verify "${positional[0]}^{commit}") || die "unknown base ${positional[0]}" 2
    head=$(git rev-parse --verify "${positional[1]:-HEAD}^{commit}") || die "unknown head ${positional[1]:-HEAD}" 2
    [[ -n $what ]] || what=$(git log --reverse --format='- %s' "$base..$head")
fi
head=$(git rev-parse "$head")
[[ $(git rev-list --count "$base..$head") -gt 0 ]] || die "no commits between ${base:0:7} and ${head:0:7}" 2
[[ -n $what ]] || die "--what is empty" 2
[[ -n $requirements ]] || requirements='Not provided. Judge the change on its own terms.'

template=${AI_REVIEW_TEMPLATE:-}
if [[ -z $template ]]; then
    while IFS= read -r candidate; do
        if [[ -f $candidate ]]; then template=$candidate; break; fi
    done < <(
        printf '%s\n' "${HOME:-}/.agents/skills/superpowers/requesting-code-review/code-reviewer.md" \
            "${HOME:-}/.codex/skills/requesting-code-review/code-reviewer.md"
        ls -1d "${HOME:-}"/.claude/plugins/cache/*/superpowers/*/skills/requesting-code-review/code-reviewer.md 2>/dev/null | sort -V -r
    )
fi
[[ -n $template && -f $template ]] \
    || die "prompt template not found: install the superpowers requesting-code-review skill or set AI_REVIEW_TEMPLATE" 2

shown_proxy=none
[[ -z ${HTTPS_PROXY:-} ]] || shown_proxy=$(sed -E 's#://[^/@]+@#://***@#' <<<"$HTTPS_PROXY")

out_dir=${AI_REVIEW_OUT_DIR:-output/ai-review/${base:0:7}-${head:0:7}}
mkdir -p "$out_dir"
prompt_file=$out_dir/prompt.md
review_file=$out_dir/review.md

tree_note="The working tree at $repo_root is checked out at the head commit, so files you read there match the diff."
if [[ $(git rev-parse HEAD) != "$head" ]] || [[ -n $(git status --porcelain --untracked-files=no) ]]; then
    tree_note="The working tree at $repo_root is NOT at the head commit. Files you read there may differ from the head; trust the diff below for changed lines."
fi

git -c core.quotePath=false diff --stat "$base" "$head" >"$out_dir/diff-stat.txt"
git -c core.quotePath=false -c diff.noprefix=false diff -M "$base" "$head" >"$out_dir/diff.patch"

python3 - "$template" "$prompt_file" "$base" "$head" "$what" "$requirements" "$tree_note" \
    "$out_dir/diff-stat.txt" "$out_dir/diff.patch" <<'PY'
import re, sys
template, out, base, head, what, reqs, tree_note, stat_path, diff_path = sys.argv[1:]
text = open(template, encoding="utf-8").read()
values = {
    "WHAT_WAS_IMPLEMENTED": what, "DESCRIPTION": what,
    "PLAN_OR_REQUIREMENTS": "the requirements below", "PLAN_REFERENCE": reqs,
    "BASE_SHA": base, "HEAD_SHA": head,
}
text = re.sub(r"\{(" + "|".join(values) + r")\}", lambda m: values[m.group(1)], text)
stat = open(stat_path, encoding="utf-8", errors="replace").read()
diff = open(diff_path, encoding="utf-8", errors="replace").read()
# Fences longer than any backtick run inside, so a diff of Markdown cannot close them early.
fence = "`" * max(3, max((len(r) for r in re.findall(r"`+", stat + diff)), default=0) + 1)
text += f"""

## Rules for this run

- This review is read-only. Do not run shell commands, git, builds, or tests, and do not edit files. Terminal access is disabled, and a request to run a command aborts the review.
- The git commands above are for reference only; the diff they would print is included below.
- You may read files in the repository to check callers and contracts. {tree_note}
- Repository rules are in AGENTS.md and CLAUDE.md at the repository root.
- Answer in the output format above. Cite file:line from the head version.

## Diff stat

{fence}
{stat}{fence}

## Full diff ({base[:7]}..{head[:7]})

{fence}diff
{diff}{fence}
"""
open(out, "w", encoding="utf-8").write(text)
PY

cmd=(acpx --cwd "$repo_root" --approve-reads --non-interactive-permissions deny --no-terminal
     --model "${AI_REVIEW_MODEL:-gemini-4-hydrogen}" --timeout "${AI_REVIEW_TIMEOUT:-1200}" --format text
     --agent "$(command -v agy) acp"
     exec --config-option "thought_level=${AI_REVIEW_THOUGHT:-high}" -f "$prompt_file")

printf 'ai-review: %s..%s, model %s (thought_level=%s), proxy %s\n' "${base:0:7}" "${head:0:7}" \
    "${AI_REVIEW_MODEL:-gemini-4-hydrogen}" "${AI_REVIEW_THOUGHT:-high}" "$shown_proxy" >&2
printf 'ai-review: template %s\n' "$template" >&2
printf 'ai-review: prompt %s\n' "$prompt_file" >&2
if [[ $dry_run -eq 1 ]]; then
    printf '%q ' "${cmd[@]}"; printf '\n'
    exit 0
fi

status=0
"${cmd[@]}" 2>&1 | tee "$review_file" || status=$?

# The review text may quote these errors, so after a clean exit look only at
# acpx's agent-error lines ("⚠ ..."), which it can print and still exit 0.
# After a failed exit, also look at the end of the output.
agent_errors=$(grep -E '^(⚠|AGY_ERROR)' "$review_file" || true)
# A turn that ended normally is a finished review: any "⚠" line before it was
# either recovered (agy retries transient API errors) or the model's own text.
last_line=$(tr -d '\r' <"$review_file" | grep -v '^[[:space:]]*$' | tail -n 1 || true)
last_line=${last_line%"${last_line##*[![:space:]]}"}
if [[ $status -eq 0 && $last_line == '[done] end_turn' ]]; then
    if [[ -n $agent_errors ]]; then
        printf 'ai-review: the review completed; %s "⚠" line(s) before it were not fatal, first: %s\n' \
            "$(wc -l <<<"$agent_errors" | tr -d ' ')" "$(head -n 1 <<<"$agent_errors")" >&2
    fi
    agent_errors=''
fi
diag=$agent_errors
[[ $status -eq 0 ]] || diag+=$'\n'$(tail -n 20 "$review_file")
hint=''
if grep -q 'User location is not supported' <<<"$diag"; then
    if [[ $shown_proxy == none ]]; then
        hint="the API rejected the request's region. This machine connects directly from an unsupported region; set AI_REVIEW_PROXY to a proxy in a supported region and rerun."
    else
        hint="the API rejected the request's region. The proxy ($shown_proxy) exits in an unsupported region; switch the proxy node (for example to the US) and rerun."
    fi
elif grep -Eq 'is not recognized as a known model|did not advertise that model' <<<"$diag"; then
    hint="agy did not offer the model. Without a working proxy agy cannot load its model list; check AI_REVIEW_PROXY and AI_REVIEW_MODEL."
elif grep -q 'PERMISSION_DENIED' <<<"$diag"; then
    hint="the model asked for a denied permission (usually a command or a write) and acpx stopped. Rerun; if it repeats, tighten the prompt."
elif [[ $status -ne 0 ]] && grep -Eqi 'timed out|timeout' <<<"$diag"; then
    hint="no answer within ${AI_REVIEW_TIMEOUT:-1200}s. Check the proxy, or raise AI_REVIEW_TIMEOUT."
elif [[ -n $agent_errors ]]; then
    hint="acpx reported an agent error: $(head -n 1 <<<"$agent_errors")"
fi
if [[ -n $hint ]]; then
    die "review failed: $hint Output: $review_file"
fi
[[ $status -eq 0 ]] || die "acpx exited with status $status. Output: $review_file" "$status"
printf 'ai-review: review saved to %s\n' "$review_file" >&2
