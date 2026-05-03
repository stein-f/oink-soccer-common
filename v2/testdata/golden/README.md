# Golden snapshots

This directory holds reference outputs for the v2 engine rebuild.

## `v1-baseline/`

JSON snapshots of v1 engine output for a pinned set of (seed, formation, lineup) inputs. **Generated, not hand-edited.**

Regenerate with:

```sh
make -C v2 snapshot-v1
# or, from repo root:
go run ./cmd/snapshot-v1
```

These are reference data — they are *not* the v2 engine's expected output. They exist so we can quantify how far v2 diverges from v1 at any point during the rebuild. They should rarely change; if they do, it means v1 itself has changed.

## `v2/` (future)

Once Phase 4 lands the new engine, this directory will hold v2's own golden snapshots — these *are* expected output and the snapshot test will fail if they drift.

## Update protocol

When a code change causes a snapshot diff:

1. Run the tests and read the diff carefully.
2. Decide: **regression** (a bug — fix the code) or **intentional** (the model changed and we expected this).
3. If intentional, regenerate the snapshots and commit them.
4. **The PR description must explain why the snapshots changed.** Reviewers should treat snapshot diffs as load-bearing — a bare "regenerated snapshots" message is not sufficient.

The point of golden tests is to make behavior changes visible. Bypassing that signal defeats the purpose.
