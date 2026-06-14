---
name: scanning-project-secrets
description: Use when asked to perform a full-project secret scan across many tracked files or multiple directories, or when systematic coverage of an entire repository for leaked credentials is required.
---

# Scanning Project Secrets

## Overview

**Large projects require parallel agent teams and explicit progress tracking.** Without structure, agents scan sequentially, miss scope, and lose track of progress. Split the codebase into groups, track with a task file, and scan in parallel.

## When to Use

- Full-repository scan (not just staged/changed files)
- More than ~30 tracked files across multiple directories
- User asks for "thorough", "complete", or "systematic" scanning

When NOT to use:
- Pre-commit check of staged files only
- Small projects (<15 files)

## Core Pattern

```
SCOPE → GROUP → TRACK → PARALLEL SCAN → SYNTHESIZE
```

## Quick Reference

| Step | Action |
|------|--------|
| 1 | `git ls-files` for exact tracked set |
| 2 | Split into 4-6 logical groups |
| 3 | Write `workspace/secret-scan-tasks.md` with checkboxes |
| 4 | Dispatch 1 agent per group in parallel |
| 5 | Update task file and produce final summary |

## Procedure

### 1. Scope and Group

```bash
git ls-files | wc -l
git ls-files
```

Split into 4-6 logical groups by directory/function. Cross-check: sum of group file counts must equal the `git ls-files` total.

### 2. Create Task File

Write `workspace/secret-scan-tasks.md` with:
- One section per group
- Checkbox list of files
- The 7 scan patterns
- Safe patterns to ignore

**Important:** Ignore self-referential matches inside the task file, this skill file, and any temporary scripts.

### 3. Parallel Scan

Dispatch one agent per group concurrently. Each agent gets the file list and runs grep for these 7 patterns:

```regex
sk-[a-zA-Z0-9]{20,}
sk-kimi-[a-zA-Z0-9]+
Bearer [a-zA-Z0-9_\-\.]{20,}
AKIA[0-9A-Z]{16}
://[^:]+:[^@]+@
-----BEGIN.*PRIVATE KEY-----
(secret|password|passwd|token|api_key)\s*[=:]\s*["'][^"']{8,}
```

**Do NOT flag:** placeholders (`"sk-test-key"`, `"sk-xxx"`), env var reads, test mocks, documentation examples, `.env` / `token.yaml` / `*.example` files, or matches in the task/skill files.

Each agent returns: files scanned, confirmed secrets, false positives ruled out, PASS/FAIL.

### 4. Synthesize

Update the task file checkboxes as agents return. Then produce a final summary with total files scanned, confirmed secrets, false positives reviewed, and overall PASS/FAIL.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| One big `git grep` and done | Use grouped agents for traceability |
| No task file | Create it before dispatching |
| Scanning `.` and hitting binaries | Use `git ls-files` only |
| Reporting mocks as secrets | Review context; skip placeholders |
| Missing file groups | Cross-check counts |
| Self-referential matches | Ignore the task file and skill file |
