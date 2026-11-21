## 1. Field mapping

- [x] 1.1 Read `ModelUsageStats`, `ChatModelMetadata`, `CortexStepGeneratorMetadata`, `CortexStepMetadata` field numbers from the `agy` binary descriptors.
- [x] 1.2 Check the mapping against local conversation files (output = thinking + response; generation usage = first step usage; no duplicate step indices).

## 2. Provider

- [x] 2.1 Protobuf wire reader for the needed fields, with tests for malformed input.
- [x] 2.2 `provider/antigravity`: discovery, read-only open (immutable, WAL copy), event and session parsing, range pruning.
- [x] 2.3 Unit tests with synthetic fixture DBs built in the test.

## 3. CLI and docs

- [x] 3.1 Blank imports and `--antigravity-dir` in `daily` and `session`.
- [x] 3.2 e2e: isolate Antigravity in existing tests; add an Antigravity fixture test.
- [x] 3.3 README / README_zh provider list and data paths.

## 4. Validation

- [x] 4.1 `make test`, `make lint`, `make build`.
- [x] 4.2 Compare `codetok daily --all --provider antigravity` totals with an independent decode of local data (300 files: totals match exactly).
