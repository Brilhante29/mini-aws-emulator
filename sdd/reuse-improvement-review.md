# Reuse Improvement Review

Project: `13 - mini-aws-emulator`

## Review Points

- [x] after scaffold
- [x] after architecture decision
- [x] after first working slice
- [x] after benchmark result
- [x] before publication
- [x] after CI failure, if applicable: no project CI failure before first publication

## Findings

| Finding | Classification | Kit Area | Action | Status |
|---|---|---|---|---|
| Kumo release tag and container tag differ, and a mutable image made the first build ambiguous. | `patch_now` | `decision-brain, docs, skills, validation` | Pin reviewed tag plus digest and reject mutable Kumo references. | completed in `cea7f9f` |
| SDK/emulator compatibility warnings can be accidentally swallowed or flood CI. | `patch_now` | `cloud guidance, benchmark contract` | Require numeric diagnostics for intentionally handled SDK warnings. | completed in `cea7f9f` |
| A reusable Go cloud-conformance package could remove code in future cloud projects. | `backlog` | `harness` | Reassess after a second project needs the same three-port runner; do not abstract from one use. | recorded |
| Project-specific S3/SQS/DynamoDB operation ports should move into the kit now. | `reject` | `project code` | Keep them here until another repository proves the same abstraction is shared. | rejected |

## Patch Now Decisions

- Updated the Kumo snapshot, release/tag rule, OCI digest, cloud skill, component pack, and project catalog.
- Added reusable validation that rejects a committed mutable Kumo image reference.
- Added provider metadata and numeric compatibility diagnostics to cloud evidence rules.
- Validated, committed, pushed, and confirmed green kit CI before project publication.

## Backlog Decisions

- Consider a language-neutral cloud parity result schema after another cloud-backed project produces compatible evidence.
- Consider extracting a Go conformance helper only after the next project reveals stable shared behavior.

## Rejected Improvements

- Do not upstream the project's cloud ports, AWS calls, or warning string.
- Do not add a generic provider plugin system; the AWS SDK endpoint switch already solves the current problem.

## Final Gate

- [x] Reusable improvements were patched or recorded.
- [x] Project-specific implementation was not moved into the kit.
- [x] Validation reflects the mutable Kumo image mistake discovered during the project.
