# EAP-003 Final Implementation

Use `final_impl_v1.md` as approved for implementation. No OpenSpec change is required because this task preserves user-facing CLI output and date semantics; the change is an internal performance optimization with focused regression tests. The task board records `Change=-` for that reason.

Implementation remains limited to:

- shared provider range-aware collection types/helpers
- command collection wiring
- provider candidate file filtering
- focused cmd/provider tests and existing e2e smoke

Reviewer constraints folded into implementation:

- provider timestamp filtering must mirror localized inclusive date-key semantics
- `session --until` must expand to the full local day before provider options are built
- file `ModTime` can only exclude candidates before `Since`; it must not exclude files after `Until`
- range-aware collection must reuse the same provider filter and directory override logic as full-history collection
