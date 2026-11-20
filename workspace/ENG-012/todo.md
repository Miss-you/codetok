# ENG-012 Open Items

- [ ] End-to-end on the runner (needs ENG-011 merged and this PR merged, which enables the workflow): this PR carries its own records comment, so its merge is the first real run. Check: `docs/changes/ENG-012/` appears on `main` with files equal to the comment's list and commit `docs: archive ENG-012 records (#<PR>) [skip ci]`; CI does not run for that commit.
- [ ] Merge a PR without a records comment: the job ends green, no commit.
- [ ] Re-run the job for the archived PR (Actions → Re-run): log says "already holds this archive", no commit.
- [ ] Invalid agy output on the runner: covered by tests only; optionally force it once with `ARCHIVE_MODEL=bogus` via a manual run.
- [ ] If `main` gets branch protection later, the direct push will fail; switch to an auto-merged PR whose head branch is excluded from AI review (#57 notes this).
