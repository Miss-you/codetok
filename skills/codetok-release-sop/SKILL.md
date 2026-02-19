---
name: codetok-release-sop
description: Cut and verify codetok releases from merged main commits through GitHub Release and npm publish. Use when asked to prepare a version bump, create/push release tags, monitor release workflow, troubleshoot publish failures, rerun failed jobs, and validate installability of @yousali/codetok.
---

# Codetok Release SOP

## Preconditions
- Release from `main` only.
- Use semantic version tags in `vX.Y.Z` format.
- Publish workflow is `.github/workflows/release.yml`.
- npm package is `@yousali/codetok`.
- Use a GitHub identity with repo write access and `gh` authenticated.

## 1) Confirm Ready-to-Release Commit
```bash
gh pr view <PR_NUMBER> --repo miss-you/codetok --json state,mergedAt,baseRefName,headRefName,url
gh api repos/miss-you/codetok/commits/main --jq '.sha[0:7] + " " + .commit.message'
```
- Require `state=MERGED` and target branch `main`.
- Use current `origin/main` commit as release source.

## 2) Sync Tags and Choose Version
```bash
git fetch origin --tags
git tag --list --sort=version:refname
```
- Pick the next unused version.
- Never reuse an existing tag.
- Bump patch for fixes, bump minor for user-facing features.

## 3) Create and Push Tag
If local `main` is clean and up to date:
```bash
git checkout main
git pull --ff-only origin main
git tag -a vX.Y.Z -m "release: vX.Y.Z"
git push origin vX.Y.Z
```

If local workspace should not be changed, tag the remote commit directly:
```bash
git fetch origin
git tag -a vX.Y.Z -m "release: vX.Y.Z" origin/main
git push origin vX.Y.Z
```

## 4) Monitor Release Workflow
```bash
gh run list --repo miss-you/codetok --workflow release.yml --limit 3
gh run watch <RUN_ID> --repo miss-you/codetok --exit-status
```
Expect both jobs to succeed:
- `Release`
- `Publish npm`

## 5) Verify GitHub Release Artifacts
```bash
gh release view vX.Y.Z --repo miss-you/codetok --json tagName,isDraft,isPrerelease,assets
```
Require:
- `isDraft=false`
- `isPrerelease=false`
- assets include (using `X.Y.Z` without leading `v` in filenames):
  - `checksums.txt`
  - `codetok_X.Y.Z_darwin_amd64.tar.gz`
  - `codetok_X.Y.Z_darwin_arm64.tar.gz`
  - `codetok_X.Y.Z_linux_amd64.tar.gz`
  - `codetok_X.Y.Z_linux_arm64.tar.gz`
  - `codetok_X.Y.Z_windows_amd64.zip`
  - `codetok_X.Y.Z_windows_arm64.zip`

## 6) Verify npm Publish and Install
```bash
npm view @yousali/codetok version
npm view @yousali/codetok dist-tags --json
npm i -g @yousali/codetok@X.Y.Z
which codetok
codetok version
```
Require:
- npm `latest` is `X.Y.Z`.
- installed binary runs and reports expected version.
- `which codetok` points to the npm-installed command, not an older local binary.

## 7) Failure Handling
### `Publish npm` fails with E403
- Cause: token lacks publish permission or wrong package namespace.
- Check repo secret presence:
```bash
gh secret list --repo miss-you/codetok
```
- Confirm workflow publishes scoped package `@yousali/codetok`.
- Rotate `NPM_TOKEN` and rerun failed jobs.

### Release job succeeds but npm package seems unavailable
- Inspect logs:
```bash
gh run view <RUN_ID> --repo miss-you/codetok --log
```
- Confirm `Publish npm` completed successfully.
- Recheck npm after a short propagation delay.

### PR blocked with `BuildExpected`
- Cause: branch protection requires `Build` check name but workflow reports only matrix check names.
- Ensure CI includes a summary check named `Build` and rerun checks.

## 8) Rerun Failed Release Attempt
```bash
gh run rerun <RUN_ID> --repo miss-you/codetok --failed
gh run watch <RUN_ID> --repo miss-you/codetok --exit-status
```
Use rerun when tag/version is correct and only credentials or transient failures were fixed.

## 9) Post-Release Record
Record these outputs:
- release tag and source commit SHA
- release run URL
- npm version and dist-tags snapshot
- install verification output (`codetok version`)
