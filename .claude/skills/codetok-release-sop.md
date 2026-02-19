# codetok Release SOP (for Claude)

## Scope
- Repository: `Miss-you/codetok`
- Workflow: `.github/workflows/release.yml`
- npm package: `@yousali/codetok`
- Release source branch: `main`

## 1. Confirm merged commit on `main`
```bash
gh pr view <PR_NUMBER> --repo Miss-you/codetok --json state,mergedAt,baseRefName,headRefName,url
gh api repos/Miss-you/codetok/commits/main --jq '.sha[0:7] + " " + .commit.message'
```
Pass criteria:
- PR is merged
- base branch is `main`

## 2. Pick next version tag
```bash
git fetch origin --tags
git tag --list --sort=version:refname
```
Rules:
- Use `vX.Y.Z`
- Never reuse existing tag
- Patch for fixes, minor for user-facing features

## 3. Create and push tag
Standard:
```bash
git checkout main
git pull --ff-only origin main
git tag -a vX.Y.Z -m "release: vX.Y.Z"
git push origin vX.Y.Z
```

Without changing local branch state:
```bash
git fetch origin
git tag -a vX.Y.Z -m "release: vX.Y.Z" origin/main
git push origin vX.Y.Z
```

## 4. Monitor release workflow
```bash
gh run list --repo Miss-you/codetok --workflow release.yml --limit 3
gh run watch <RUN_ID> --repo Miss-you/codetok --exit-status
```
Expected jobs:
- `Release`
- `Publish npm`

## 5. Verify GitHub release artifacts
```bash
gh release view vX.Y.Z --repo Miss-you/codetok --json tagName,isDraft,isPrerelease,assets
```
Must include:
- `checksums.txt`
- `codetok_<version>_darwin_amd64.tar.gz`
- `codetok_<version>_darwin_arm64.tar.gz`
- `codetok_<version>_linux_amd64.tar.gz`
- `codetok_<version>_linux_arm64.tar.gz`
- `codetok_<version>_windows_amd64.zip`
- `codetok_<version>_windows_arm64.zip`

## 6. Verify npm publish and install
```bash
npm view @yousali/codetok version
npm view @yousali/codetok dist-tags --json
npm i -g @yousali/codetok@X.Y.Z
which codetok
codetok version
```
Pass criteria:
- `latest` equals `X.Y.Z`
- installed binary runs and shows expected version

## 7. Common failure handling
### `E403` on npm publish
- Check `NPM_TOKEN` exists in repo secrets:
```bash
gh secret list --repo Miss-you/codetok
```
- Verify package namespace is `@yousali/codetok`
- Rotate token and rerun failed jobs if needed

### Release succeeded but npm not visible
- Inspect workflow logs:
```bash
gh run view <RUN_ID> --repo Miss-you/codetok --log
```
- Wait short time for npm propagation and recheck

### PR shows `BuildExpected` pending
- Branch protection expects a check named `Build`
- Ensure CI reports a summary check named `Build`

## 8. Rerun a failed release run
```bash
gh run rerun <RUN_ID> --repo Miss-you/codetok --failed
gh run watch <RUN_ID> --repo Miss-you/codetok --exit-status
```

## 9. Post-release record
Keep these records:
- release tag and source commit SHA
- release workflow run URL
- npm version and dist-tags snapshot
- `codetok version` output from install verification
